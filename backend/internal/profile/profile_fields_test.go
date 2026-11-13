package profile_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/profile"
	"github.com/w3con/sifty/internal/store"
)

// TestLegacyProfileSurvivesSectionSaveRoundTrip catches silent data loss during the contract
// migration: a save from an older or newer editor must not erase retained Headline, skill context,
// per-experience skills, day rates, hours, work limits, free-form locations, notice/timezone,
// demographics, documents or facts. It also proves a changed visible section uses the new values
// while hidden legacy values remain stored.
func TestLegacyProfileSurvivesSectionSaveRoundTrip(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "profile.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	legacy := `{
		"identity": {"displayName":"Legacy Owner","legalName":"Legal Owner","headline":"Legacy Headline"},
		"demographics": {"gender":"woman","race":["asian"],"ethnicity":"hispanic_or_latino"},
		"skills": [
			{"key":"go","name":"Go","level":"Production","context":"production systems","provenance":{"kind":"document","reference":"doc-old","quote":"Go"}},
			{"key":"rust","name":"Rust","level":"Learning","context":"side projects"}
		],
		"experience": [
			{"company":"Acme","title":"Engineer","start":"01/2020","end":"","summary":"Legacy plain summary","skills":["Go","Rust"]}
		],
		"targetRoles": ["Backend Engineer"],
		"targetIndustries": ["Software"],
		"compensation": {
			"cashMin":50000,"cashTarget":70000,"currency":"EUR","grossNet":"gross","period":"year",
			"dayRates":{"min":400,"target":500,"currency":"EUR","period":"day"},"equityOnTop":true
		},
		"engagement": {"types":["employee","contractor"]},
		"hours": {"mode":"full_time","weekly":20},
		"workModes": {"modes":["remote"],"limits":["No night shifts"]},
		"geography": {
			"current":"Paris, France","desired":["Canada"],"eligibility":["France"],
			"relocation":["Europe"],"willingToRelocate":true,"sponsorship":"required","timezones":["Europe/Paris"]
		},
		"languages": [{"name":"French","level":"Fluent-ish"}],
		"availability": {"noticePeriod":"2 months","startDate":"March 2027","timezone":"Europe/Paris"},
		"dealbreakers": ["night shifts"]
	}`
	if _, err := s.CreateProfile(profile.LocalProfileID, legacy); err != nil {
		t.Fatalf("seed legacy profile: %v", err)
	}

	// A real document and proposed fact must remain untouched by profile section writes.
	docID := "doc-legacy"
	if err := s.CreateProfileDocument(&store.ProfileDocumentRecord{
		ID: docID, ProfileID: profile.LocalProfileID, Kind: "md", OriginalName: "cv.md",
		ProcessingState: "analyzing", ExtractedText: "Legacy headline",
	}); err != nil {
		t.Fatalf("create document: %v", err)
	}
	if err := s.StartProfileDocumentAnalysis(docID, `{"runId":"run-legacy"}`); err != nil {
		t.Fatalf("start document analysis: %v", err)
	}
	fact := &store.ProfileFactRecord{
		ID: "fact-legacy", ProfileID: profile.LocalProfileID, DocumentID: &docID,
		FieldKey: "identity.headline", ValueJSON: `"Legacy headline"`,
		Quote: "Legacy headline", Origin: "document", State: profile.FactProposed,
	}
	published, err := s.CompleteProfileDocumentAnalysis(
		docID, "run-legacy", "analyzed", `{"runId":"run-legacy"}`, "", []*store.ProfileFactRecord{fact})
	if err != nil || !published {
		t.Fatalf("complete document analysis: published=%v err=%v", published, err)
	}

	svc := profile.NewService(s, nil)
	if _, err := svc.Get(); err != nil {
		t.Fatalf("read seeded profile: %v", err)
	}

	// An older Identity editor sends only the fields it knows. It must not erase LegalName or the
	// separate demographics object.
	afterIdentity, changed, err := svc.UpdateSection("identity", json.RawMessage(
		`{"displayName":"Legacy Owner 2","headline":"Engineer"}`))
	if err != nil || !changed {
		t.Fatalf("identity section: changed=%v err=%v", changed, err)
	}
	if afterIdentity.Data.Identity.LegalName != "Legal Owner" {
		t.Errorf("identity save lost legal name: %+v", afterIdentity.Data.Identity)
	}
	if afterIdentity.Data.Demographics.Gender != "woman" || len(afterIdentity.Data.Demographics.Race) != 1 {
		t.Errorf("identity save lost demographics: %+v", afterIdentity.Data.Demographics)
	}

	// A current editor omits the removed Context field and the old levels. The server keeps the
	// old context text and the untouched old level, and accepts the new explicit level.
	afterSkills, changed, err := svc.UpdateSection("skills", json.RawMessage(
		`[{"key":"go","name":"Go","level":"Proficient"},{"key":"rust","name":"Rust","level":"Learning"}]`))
	if err != nil || !changed {
		t.Fatalf("skills section: changed=%v err=%v", changed, err)
	}
	if afterSkills.Data.Skills[0].Level != profile.SkillProficient || afterSkills.Data.Skills[0].Context != "production systems" {
		t.Errorf("skill save lost context or ignored explicit level: %+v", afterSkills.Data.Skills[0])
	}
	if afterSkills.Data.Skills[1].Level != profile.SkillLearning || afterSkills.Data.Skills[1].Context != "side projects" {
		t.Errorf("legacy skill level/context lost: %+v", afterSkills.Data.Skills[1])
	}
	goSkillID := afterSkills.Data.Skills[0].ID
	if goSkillID == "" {
		t.Error("skill row has no stable id")
	}
	// Rename one skill by its stable id and add a different one. The renamed row keeps its hidden
	// context; the new row must not inherit a deleted row's context.
	afterSkillRename, changed, err := svc.UpdateSection("skills", json.RawMessage(
		`[{"id":"`+goSkillID+`","key":"go","name":"GoLang","level":"Proficient"},{"key":"python","name":"Python","level":"Beginner"}]`))
	if err != nil || !changed {
		t.Fatalf("skill rename section: changed=%v err=%v", changed, err)
	}
	if afterSkillRename.Data.Skills[0].Name != "GoLang" || afterSkillRename.Data.Skills[0].Context != "production systems" {
		t.Errorf("renamed skill lost context: %+v", afterSkillRename.Data.Skills[0])
	}
	if afterSkillRename.Data.Skills[1].Context != "" {
		t.Errorf("new skill inherited another row's context: %+v", afterSkillRename.Data.Skills[1])
	}

	richSummary := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Rich summary"}]}]}`
	experienceValue, _ := json.Marshal([]map[string]any{{
		"company": "Acme", "title": "Engineer", "start": "01/2020", "current": true,
		"summary": "Stale plain text", "summaryDocument": json.RawMessage(richSummary),
	}})
	afterExperience, changed, err := svc.UpdateSection("experience", experienceValue)
	if err != nil || !changed {
		t.Fatalf("experience section: changed=%v err=%v", changed, err)
	}
	role := afterExperience.Data.Experience[0]
	if !role.Current || role.End != "" {
		t.Errorf("current role kept an acting end date: %+v", role)
	}
	if role.Summary != "Rich summary" {
		t.Errorf("rich summary did not derive authoritative plain text: %q", role.Summary)
	}
	if len(role.Skills) != 2 || role.Skills[0] != "Go" || role.Skills[1] != "Rust" {
		t.Errorf("experience save lost legacy per-role skills: %+v", role.Skills)
	}
	acmeID := role.ID
	if acmeID == "" {
		t.Error("experience row has no stable id")
	}
	// Rename and reorder by stable id: the hidden skills and rich summary follow the same row.
	renameValue, _ := json.Marshal([]map[string]any{
		{"id": acmeID, "company": "Acme", "title": "Senior Engineer", "start": "01/2020", "current": true},
		{"company": "Beta", "title": "Manager", "start": "02/2021", "summary": "New role", "skills": []string{}},
	})
	afterRename, changed, err := svc.UpdateSection("experience", renameValue)
	if err != nil || !changed {
		t.Fatalf("experience rename section: changed=%v err=%v", changed, err)
	}
	var renamedAcme, beta profile.Experience
	for _, row := range afterRename.Data.Experience {
		switch row.ID {
		case acmeID:
			renamedAcme = row
		case "":
			t.Errorf("new experience row did not receive an id: %+v", row)
		default:
			if row.Company == "Beta" {
				beta = row
			}
		}
	}
	if renamedAcme.Title != "Senior Engineer" || len(renamedAcme.Skills) != 2 || renamedAcme.Summary != "Rich summary" {
		t.Errorf("renamed role lost hidden details: %+v", renamedAcme)
	}
	if len(beta.Skills) != 0 || beta.Summary != "New role" {
		t.Errorf("new role inherited a deleted row's hidden details: %+v", beta)
	}
	reorderValue, _ := json.Marshal([]map[string]any{
		{"id": beta.ID, "company": "Beta", "title": "Manager", "start": "02/2021", "summary": "New role"},
		{"id": acmeID, "company": "Acme", "title": "Senior Engineer", "start": "01/2020", "current": true},
	})
	afterReorder, changed, err := svc.UpdateSection("experience", reorderValue)
	if err != nil || !changed {
		t.Fatalf("experience reorder section: changed=%v err=%v", changed, err)
	}
	if afterReorder.Data.Experience[0].ID != beta.ID || len(afterReorder.Data.Experience[0].Skills) != 0 {
		t.Errorf("reorder attached hidden data to Beta: %+v", afterReorder.Data.Experience[0])
	}
	if afterReorder.Data.Experience[1].ID != acmeID || len(afterReorder.Data.Experience[1].Skills) != 2 {
		t.Errorf("reorder lost Acme hidden data: %+v", afterReorder.Data.Experience[1])
	}
	// Delete Acme and add Gamma in the same save. Gamma must not inherit Acme's hidden fields.
	deleteValue, _ := json.Marshal([]map[string]any{
		{"company": "Gamma", "title": "Lead", "start": "03/2022"},
	})
	afterDelete, changed, err := svc.UpdateSection("experience", deleteValue)
	if err != nil || !changed {
		t.Fatalf("experience delete section: changed=%v err=%v", changed, err)
	}
	if len(afterDelete.Data.Experience) != 1 || afterDelete.Data.Experience[0].Company != "Gamma" {
		t.Fatalf("experience delete result = %+v", afterDelete.Data.Experience)
	}
	if len(afterDelete.Data.Experience[0].Skills) != 0 || afterDelete.Data.Experience[0].Summary != "" {
		t.Errorf("new role inherited deleted row's hidden details: %+v", afterDelete.Data.Experience[0])
	}

	afterComp, changed, err := svc.UpdateSection("compensation", json.RawMessage(
		`{"cashMin":60000,"cashTarget":80000,"currency":"USD","grossNet":"gross","period":"year","crypto":"crypto_only","equityOnTop":false}`))
	if err != nil || !changed {
		t.Fatalf("compensation section: changed=%v err=%v", changed, err)
	}
	if afterComp.Data.Compensation.DayRates.Min == nil || *afterComp.Data.Compensation.DayRates.Min != 400 {
		t.Errorf("compensation save lost legacy day rates: %+v", afterComp.Data.Compensation.DayRates)
	}
	if afterComp.Data.Compensation.Crypto != profile.PaymentCryptoOnly {
		t.Errorf("compensation crypto = %q, want crypto_only", afterComp.Data.Compensation.Crypto)
	}

	afterWork, changed, err := svc.UpdateSection("work", json.RawMessage(
		`{"engagement":{"types":["employment","contract"]},"hours":{"mode":"part_time","weeklyMin":10,"weeklyMax":20},"workModes":{"modes":["hybrid"]}}`))
	if err != nil || !changed {
		t.Fatalf("work section: changed=%v err=%v", changed, err)
	}
	if afterWork.Data.Hours.Weekly == nil || *afterWork.Data.Hours.Weekly != 20 {
		t.Errorf("work save lost stored weekly hours: %+v", afterWork.Data.Hours)
	}
	if len(afterWork.Data.WorkModes.Limits) != 1 || afterWork.Data.WorkModes.Limits[0] != "No night shifts" {
		t.Errorf("work save lost legacy work limits: %+v", afterWork.Data.WorkModes.Limits)
	}
	if afterWork.Data.Engagement.Types[0] != profile.EngagementEmployment {
		t.Errorf("visible engagement type was not updated: %+v", afterWork.Data.Engagement.Types)
	}
	// The current part-time editor exposes Weekly as desired hours. It remains effective after a
	// range is cleared; only the range itself is removed. This is the current UI meaning, not the
	// older "legacy weekly is suppressed by the range marker" compatibility rule.
	afterWorkHours := profile.ContextPreferences(afterWork)["hours"].(profile.Hours)
	if afterWorkHours.Weekly == nil || *afterWorkHours.Weekly != 20 {
		t.Errorf("part-time context dropped desired weekly hours: %+v", afterWorkHours)
	}
	if afterWorkHours.WeeklyMin == nil || *afterWorkHours.WeeklyMin != 10 || afterWorkHours.WeeklyMax == nil || *afterWorkHours.WeeklyMax != 20 {
		t.Errorf("part-time context dropped the optional range: %+v", afterWorkHours)
	}
	afterWorkClear, changed, err := svc.UpdateSection("work", json.RawMessage(
		`{"hours":{"mode":"part_time","weeklyMin":null,"weeklyMax":null}}`))
	if err != nil || !changed {
		t.Fatalf("work clear range section: changed=%v err=%v", changed, err)
	}
	reloadedWork, err := svc.Get()
	if err != nil {
		t.Fatalf("reload after work range clear: %v", err)
	}
	if reloadedWork.Data.Hours.Weekly == nil || *reloadedWork.Data.Hours.Weekly != 20 {
		t.Errorf("work range clear lost stored weekly hours: %+v", reloadedWork.Data.Hours)
	}
	prefsHours := profile.ContextPreferences(reloadedWork)["hours"].(profile.Hours)
	if prefsHours.Weekly == nil || *prefsHours.Weekly != 20 {
		t.Errorf("part-time context lost desired weekly hours after range clear: %+v", prefsHours)
	}
	if prefsHours.WeeklyMin != nil || prefsHours.WeeklyMax != nil {
		t.Errorf("cleared part-time range remained active: %+v", prefsHours)
	}
	// An unset working time uses the same UI meaning as full-time: stored part-time values are
	// retained but are not effective inputs.
	unsetMode := *reloadedWork
	unsetMode.Data.Hours.Mode = ""
	if unsetHours := profile.ContextPreferences(&unsetMode)["hours"].(profile.Hours); unsetHours.Weekly != nil || unsetHours.WeeklyMin != nil || unsetHours.WeeklyMax != nil {
		t.Errorf("unset working time made stored hours active: %+v", unsetHours)
	}
	_ = afterWorkClear

	afterGeography, changed, err := svc.UpdateSection("geography", json.RawMessage(
		`{"currentLocation":{"country":"FR","city":"Paris"},"preferredLocations":[{"country":"CA"}],"workAuthorizations":[{"country":"FR","status":"eligible"}],"timezoneIds":["Europe/Paris"],"desiredOverlapHours":3}`))
	if err != nil || !changed {
		t.Fatalf("geography section: changed=%v err=%v", changed, err)
	}
	if afterGeography.Data.Geography.Current != "Paris, France" ||
		len(afterGeography.Data.Geography.Desired) != 1 ||
		len(afterGeography.Data.Geography.Eligibility) != 1 ||
		len(afterGeography.Data.Geography.Timezones) != 1 {
		t.Errorf("geography save lost legacy location/authorization/timezone values: %+v", afterGeography.Data.Geography)
	}
	// Explicitly clearing the current typed geography must not revive retained legacy strings.
	afterGeographyClear, changed, err := svc.UpdateSection("geography", json.RawMessage(
		`{"currentLocation":null,"preferredLocations":[],"workAuthorizations":[],"timezoneIds":[]}`))
	if err != nil || !changed {
		t.Fatalf("geography clear section: changed=%v err=%v", changed, err)
	}
	reloadedGeo, err := svc.Get()
	if err != nil {
		t.Fatalf("reload after geography clear: %v", err)
	}
	if reloadedGeo.Data.Geography.Current != "Paris, France" || len(reloadedGeo.Data.Geography.Desired) != 1 {
		t.Errorf("geography clear deleted retained legacy values: %+v", reloadedGeo.Data.Geography)
	}
	effectiveGeo := profile.ContextPreferences(reloadedGeo)["geography"].(profile.Geography)
	if effectiveGeo.Current != "" || len(effectiveGeo.Desired) != 0 || len(effectiveGeo.Eligibility) != 0 || len(effectiveGeo.Timezones) != 0 {
		t.Errorf("cleared geography revived legacy restrictions: %+v", effectiveGeo)
	}
	if profile.ContextPreferences(reloadedGeo)["availability"].(profile.Availability).Timezone != "" {
		t.Error("cleared timezones revived the legacy availability timezone")
	}
	_ = afterGeographyClear

	afterAvailability, changed, err := svc.UpdateSection("availability", json.RawMessage(
		`{"noticeValue":6,"noticeUnit":"week","startDate":"01/02/2027"}`))
	if err != nil || !changed {
		t.Fatalf("availability section: changed=%v err=%v", changed, err)
	}
	if afterAvailability.Data.Availability.NoticePeriod != "2 months" ||
		afterAvailability.Data.Availability.Timezone != "Europe/Paris" {
		t.Errorf("availability save lost legacy notice/timezone: %+v", afterAvailability.Data.Availability)
	}
	afterNoticeClear, changed, err := svc.UpdateSection("availability", json.RawMessage(
		`{"noticeValue":null,"noticeUnit":""}`))
	if err != nil || !changed {
		t.Fatalf("availability clear section: changed=%v err=%v", changed, err)
	}
	reloadedNotice, err := svc.Get()
	if err != nil {
		t.Fatalf("reload after notice clear: %v", err)
	}
	if reloadedNotice.Data.Availability.NoticePeriod != "2 months" {
		t.Errorf("notice clear deleted retained legacy notice: %+v", reloadedNotice.Data.Availability)
	}
	if profile.ContextPreferences(reloadedNotice)["availability"].(profile.Availability).NoticePeriod != "" {
		t.Error("cleared structured notice revived the legacy notice text")
	}
	_ = afterNoticeClear

	otherDoc := `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Other requirement"}]}]}`
	otherValue, _ := json.Marshal(map[string]any{
		"text": "stale", "document": json.RawMessage(otherDoc),
	})
	afterOther, changed, err := svc.UpdateSection("other-requirements", otherValue)
	if err != nil || !changed {
		t.Fatalf("other requirements section: changed=%v err=%v", changed, err)
	}
	if afterOther.Data.OtherRequirements.Text != "Other requirement" {
		t.Errorf("other requirements plain text = %q", afterOther.Data.OtherRequirements.Text)
	}
	if len(afterOther.Data.Dealbreakers) != 1 || len(afterOther.Data.WorkModes.Limits) != 1 {
		t.Errorf("other requirements save deleted retained legacy values: %+v / %+v",
			afterOther.Data.Dealbreakers, afterOther.Data.WorkModes.Limits)
	}
	// Clearing the current block must keep the legacy values stored but must not reactivate them
	// in effective fit inputs.
	afterOtherClear, changed, err := svc.UpdateSection("other-requirements", json.RawMessage(`{"text":"","document":null}`))
	if err != nil || !changed {
		t.Fatalf("other requirements clear section: changed=%v err=%v", changed, err)
	}
	_ = afterOtherClear
	reloadedOther, err := svc.Get()
	if err != nil {
		t.Fatalf("reload after other requirements clear: %v", err)
	}
	if len(reloadedOther.Data.Dealbreakers) != 1 || len(reloadedOther.Data.WorkModes.Limits) != 1 {
		t.Errorf("other requirements clear deleted retained legacy values: %+v / %+v",
			reloadedOther.Data.Dealbreakers, reloadedOther.Data.WorkModes.Limits)
	}
	clearedPrefs := profile.ContextPreferences(reloadedOther)
	if len(clearedPrefs["dealbreakers"].([]string)) != 0 {
		t.Errorf("cleared other requirements revived legacy dealbreakers: %v", clearedPrefs["dealbreakers"])
	}
	if len(clearedPrefs["workModes"].(profile.WorkModes).Limits) != 0 {
		t.Errorf("cleared other requirements revived legacy work limits: %+v", clearedPrefs["workModes"])
	}
	if clearedPrefs["otherRequirements"].(profile.OtherRequirements).Text != "" {
		t.Errorf("cleared other requirements kept text: %+v", clearedPrefs["otherRequirements"])
	}

	afterDemographics, changed, err := svc.UpdateSection("demographics", json.RawMessage(`{"gender":"prefer_not_to_say"}`))
	if err != nil || !changed {
		t.Fatalf("demographics section: changed=%v err=%v", changed, err)
	}
	if afterDemographics.Data.Demographics.Gender != profile.GenderPreferNotSay ||
		len(afterDemographics.Data.Demographics.Race) != 1 ||
		afterDemographics.Data.Demographics.Ethnicity != profile.EthnicityHispanic {
		t.Errorf("demographics save lost retained answers: %+v", afterDemographics.Data.Demographics)
	}
	_, changed, err = svc.UpdateSection("demographics", json.RawMessage(
		`{"gender":"self_describe","genderDescription":"Woman with a non-binary identity","race":["self_describe"],"raceDescription":"Mixed heritage"}`))
	if err != nil || !changed {
		t.Fatalf("self-describe demographics section: changed=%v err=%v", changed, err)
	}
	reloadedSelf, err := svc.Get()
	if err != nil {
		t.Fatalf("reload self-describe demographics: %v", err)
	}
	if reloadedSelf.Data.Demographics.GenderDescription != "Woman with a non-binary identity" ||
		reloadedSelf.Data.Demographics.RaceDescription != "Mixed heritage" {
		t.Errorf("self-describe text was not retained: %+v", reloadedSelf.Data.Demographics)
	}
	if profile.ContextPreferences(reloadedSelf)["demographics"] != nil {
		t.Error("self-describe text leaked into fit preferences")
	}
	if strings.Contains(strings.ToLower(profile.ContextSource(reloadedSelf).Text), "mixed heritage") {
		t.Error("self-describe text leaked into confirmed-profile evidence")
	}

	// A repeated identity payload is a normalized no-op: no revision and no derived refresh.
	beforeRevision := reloadedSelf.Revision
	noOp, changed, err := svc.UpdateSection("identity", json.RawMessage(
		`{"displayName":"Legacy Owner 2","headline":"Engineer"}`))
	if err != nil || changed {
		t.Fatalf("no-op identity: changed=%v err=%v", changed, err)
	}
	if noOp.Revision != beforeRevision {
		t.Errorf("no-op wrote revision %d, want %d", noOp.Revision, beforeRevision)
	}

	docs, err := s.ListProfileDocuments(profile.LocalProfileID)
	if err != nil || len(docs) != 1 {
		t.Errorf("documents after section saves = %d, err=%v", len(docs), err)
	}
	facts, err := s.ListProfileFacts(profile.LocalProfileID, "")
	if err != nil || len(facts) != 1 || facts[0].State != profile.FactProposed {
		t.Errorf("facts after section saves = %+v, err=%v", facts, err)
	}
}

// TestDemographicsExcludedFromFitAndExtraction catches the hidden privacy failure where optional
// gender/race/ethnicity leak through the shared candidate-context preference object, the
// confirmed-profile evidence text, the fast-rank query or document-derived facts. The only
// declared use is the application-form pathway; these pure functions make no model call.
func TestDemographicsExcludedFromFitAndExtraction(t *testing.T) {
	weekly := 20.0
	end := "12/2020"
	d := profile.ProfileData{
		Identity: profile.Identity{DisplayName: "Fit Name", LegalName: "Legal Name", Headline: "Fit Headline"},
		Demographics: profile.Demographics{
			Gender:            profile.GenderSelfDescribe,
			GenderDescription: "self-describe-gender-sentinel",
			Race:              []string{profile.RaceAsian, profile.RaceSelfDescribe},
			RaceDescription:   "self-describe-race-sentinel",
			Ethnicity:         profile.EthnicityHispanic,
		},
		TargetRoles:      []string{"Backend Engineer"},
		Skills:           []profile.Skill{{Key: "go", Name: "Go", Level: profile.SkillProficient}},
		TargetIndustries: []string{"Software"},
		Experience: []profile.Experience{{
			Company: "Acme", Title: "Engineer", End: end, Current: true, Summary: "Plain summary", Skills: []string{"Go"},
		}},
		Compensation: profile.Compensation{CashMin: floatPtr(50000), Currency: "EUR"},
		Engagement:   profile.Engagement{Types: []string{profile.EngagementEmployment}},
		Hours:        profile.Hours{Mode: profile.HoursFullTime, Weekly: &weekly},
		WorkModes:    profile.WorkModes{Modes: []string{profile.WorkRemote}, Limits: []string{"legacy limit"}},
		Geography: profile.Geography{
			Current:         "Legacy City",
			Desired:         []string{"Legacy Desired"},
			Eligibility:     []string{"Legacy Eligibility"},
			Sponsorship:     profile.SponsorshipRequired,
			Timezones:       []string{"Legacy/Timezone"},
			CurrentLocation: &profile.Location{Country: "FR", City: "Paris"},
			PreferredLocations: []profile.Location{
				{Country: "CA"},
			},
			WorkAuthorizations:  []profile.WorkAuthorization{{Country: "FR", Status: profile.AuthorizationEligible}},
			TimezoneIDs:         []string{"Europe/Paris"},
			DesiredOverlapHours: floatPtr(3),
		},
		Languages:    []profile.Language{{Name: "French", Level: profile.LanguageB2}},
		Availability: profile.Availability{NoticePeriod: "legacy notice", StartDate: "01/01/2027", Timezone: "Legacy/Timezone", NoticeValue: floatPtr(2), NoticeUnit: profile.NoticeUnitMonth},
		Dealbreakers: []string{"legacy dealbreaker"},
		OtherRequirements: profile.OtherRequirements{
			Text: "Current requirements",
		},
	}
	p := &profile.Profile{ID: profile.LocalProfileID, Revision: 7, Data: d}

	prefs := profile.ContextPreferences(p)
	prefsJSON, err := json.Marshal(prefs)
	if err != nil {
		t.Fatalf("marshal preferences: %v", err)
	}
	if _, ok := prefs["demographics"]; ok {
		t.Error("context preferences expose a demographics key")
	}
	prefsText := strings.ToLower(string(prefsJSON))
	for _, sentinel := range []string{profile.GenderSelfDescribe, "self-describe-gender-sentinel", profile.RaceAsian, profile.RaceSelfDescribe, "self-describe-race-sentinel", profile.EthnicityHispanic} {
		if strings.Contains(prefsText, strings.ToLower(sentinel)) {
			t.Errorf("fit preferences contain demographic value %q: %s", sentinel, prefsText)
		}
	}

	source := profile.ContextSource(p)
	if strings.Contains(strings.ToLower(source.Text), "demographics") {
		t.Errorf("confirmed-profile evidence exposes a demographics shape: %s", source.Text)
	}
	for _, sentinel := range []string{profile.GenderSelfDescribe, "self-describe-gender-sentinel", profile.RaceAsian, profile.RaceSelfDescribe, "self-describe-race-sentinel", profile.EthnicityHispanic} {
		if strings.Contains(strings.ToLower(source.Text), strings.ToLower(sentinel)) {
			t.Errorf("confirmed-profile evidence contains demographic value %q", sentinel)
		}
	}

	fast := profile.FastRankingQuery(nil, &d)
	for _, sentinel := range []string{profile.GenderSelfDescribe, "self-describe-gender-sentinel", profile.RaceAsian, profile.RaceSelfDescribe, "self-describe-race-sentinel", profile.EthnicityHispanic} {
		if strings.Contains(strings.ToLower(fast), strings.ToLower(sentinel)) {
			t.Errorf("fast-rank query contains demographic value %q: %s", sentinel, fast)
		}
	}

	if got := profile.FieldUses("demographics.gender"); len(got) != 1 || got[0] != profile.UseAnswerForm {
		t.Errorf("gender uses = %v, want only answer_form", got)
	}
	if got := profile.FieldUses("demographics.race"); len(got) != 1 || got[0] != profile.UseAnswerForm {
		t.Errorf("race uses = %v, want only answer_form", got)
	}
	if got := profile.FieldUses("demographics.ethnicity"); len(got) != 1 || got[0] != profile.UseAnswerForm {
		t.Errorf("ethnicity uses = %v, want only answer_form", got)
	}
	if got := profile.FieldUses("demographics.genderDescription"); len(got) != 1 || got[0] != profile.UseAnswerForm {
		t.Errorf("gender description uses = %v, want only answer_form", got)
	}
	if got := profile.FieldUses("demographics.raceDescription"); len(got) != 1 || got[0] != profile.UseAnswerForm {
		t.Errorf("race description uses = %v, want only answer_form", got)
	}
	for _, key := range profile.AllowedFactFieldKeys() {
		if strings.HasPrefix(key, "demographics.") {
			t.Errorf("document extraction allows demographic field %q", key)
		}
	}
	if _, err := profile.ValidateFactValue("demographics.gender", json.RawMessage(`"woman"`)); err == nil {
		t.Error("demographics were accepted as a document-derived fact")
	}
	if _, err := profile.ValidateFactValue("demographics.genderDescription", json.RawMessage(`"text"`)); err == nil {
		t.Error("self-describe text was accepted as a document-derived fact")
	}

	// Effective consumer mapping: full-time ignores stored part-time hours; current roles have no
	// acting end; new fields replace legacy duplicates only in the context projection.
	hours := prefs["hours"].(profile.Hours)
	if hours.Weekly != nil {
		t.Errorf("full-time context kept weekly hours: %+v", hours)
	}
	if d.Hours.Weekly == nil || *d.Hours.Weekly != weekly {
		t.Errorf("stored weekly hours were lost: %+v", d.Hours)
	}
	experience := prefs["experience"].([]profile.Experience)
	if experience[0].End != "" {
		t.Errorf("current role context kept acting end: %+v", experience[0])
	}
	if d.Experience[0].End != end {
		t.Errorf("stored current-role end was changed: %q", d.Experience[0].End)
	}
	if len(prefs["dealbreakers"].([]string)) != 0 {
		t.Errorf("legacy dealbreakers duplicated alongside other requirements: %v", prefs["dealbreakers"])
	}
	work := prefs["workModes"].(profile.WorkModes)
	if len(work.Limits) != 0 {
		t.Errorf("legacy work limits duplicated alongside other requirements: %v", work.Limits)
	}
	if d.Dealbreakers[0] != "legacy dealbreaker" || d.WorkModes.Limits[0] != "legacy limit" {
		t.Error("stored legacy requirements were mutated")
	}
	geo := prefs["geography"].(profile.Geography)
	if geo.Current != "" || len(geo.Desired) != 0 || len(geo.Eligibility) != 0 || len(geo.Timezones) != 0 {
		t.Errorf("context kept legacy geography duplicates: %+v", geo)
	}
	if d.Geography.Current != "Legacy City" || len(d.Geography.Desired) != 1 {
		t.Error("stored legacy geography was mutated")
	}
	availability := prefs["availability"].(profile.Availability)
	if availability.NoticePeriod != "" {
		t.Errorf("context kept legacy notice duplicate: %+v", availability)
	}
	if d.Availability.NoticePeriod != "legacy notice" {
		t.Error("stored legacy notice was mutated")
	}
}

// TestRichTextDocumentDerivesPlainText catches an unsupported document or HTML/embed being stored
// as if it were usable text, and proves the plain-text derivative is deterministic.
func TestRichTextDocumentDerivesPlainText(t *testing.T) {
	doc := json.RawMessage(`{"type":"doc","content":[
		{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Role"}]},
		{"type":"paragraph","content":[{"type":"text","text":"Built ","marks":[{"type":"bold"}]},{"type":"text","text":"systems"}]},
		{"type":"paragraph","content":[{"type":"text","text":"Line one"},{"type":"hardBreak"},{"type":"text","text":"Line two"}]},
		{"type":"paragraph","content":[{"type":"text","text":"First paragraph"}]},
		{"type":"paragraph","content":[{"type":"text","text":"Second paragraph"}]},
		{"type":"bulletList","content":[{"type":"listItem","content":[
			{"type":"paragraph","content":[{"type":"text","text":"Go"}]},
			{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"Nested child"}]}]}]}
		]}]}
	]}`)
	text, err := profile.RichTextPlainText(doc)
	if err != nil {
		t.Fatalf("derive plain text: %v", err)
	}
	if !strings.Contains(text, "Role") || !strings.Contains(text, "Built systems") || !strings.Contains(text, "- Go") {
		t.Errorf("plain text = %q", text)
	}
	if !strings.Contains(text, "Line one\nLine two") || strings.Contains(text, "Line oneLine two") {
		t.Errorf("hard break was dropped or concatenated: %q", text)
	}
	if !strings.Contains(text, "First paragraph\n\nSecond paragraph") || strings.Contains(text, "First paragraphSecond paragraph") {
		t.Errorf("paragraph boundary was dropped: %q", text)
	}
	if !strings.Contains(text, "- Go\n  - Nested child") || strings.Contains(text, "GoNested child") {
		t.Errorf("nested list boundary was dropped: %q", text)
	}

	data := profile.ProfileData{
		Experience:        []profile.Experience{{Company: "Acme", Title: "Engineer", Summary: "old", SummaryDocument: doc}},
		OtherRequirements: profile.OtherRequirements{Text: "old", Document: doc},
	}
	if err := data.SyncPlainText(); err != nil {
		t.Fatalf("sync plain text: %v", err)
	}
	if data.Experience[0].Summary == "old" || data.OtherRequirements.Text == "old" {
		t.Error("rich document did not replace the plain-text derivative")
	}

	invalid := json.RawMessage(`{"type":"doc","content":[{"type":"embed","attrs":{"src":"x"}}]}`)
	if _, err := profile.RichTextPlainText(invalid); err == nil {
		t.Error("unsupported embed node was accepted")
	}
	if _, err := profile.RichTextPlainText(json.RawMessage(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"<script>x</script>"}]}]}`)); err != nil {
		t.Errorf("plain text node should remain text data, got %v", err)
	}
}

func floatPtr(v float64) *float64 { return &v }
