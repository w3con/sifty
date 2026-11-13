package inspect

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/w3con/sifty/internal/worker"
)

const fixtureForm = `<!doctype html><html lang="en"><head><title>Apply</title></head><body>
<form>
 <p><label for="name">Full name</label><input id="name" name="name" type="text" maxlength="100" required></p>
 <p><label for="cover">Cover letter</label><textarea id="cover" name="cover"></textarea></p>
 <fieldset><legend>Willing to relocate?</legend>
  <label><input type="radio" name="relocate" value="yes"> Yes</label>
  <label><input type="radio" name="relocate" value="no"> No</label>
 </fieldset>
 <p><label for="cv">Upload CV</label><input id="cv" name="cv" type="file"></p>
</form>
</body></html>`

// fixtureAshbyPage simulates an ATS landing page (Ashby style) where the application form is
// hidden behind an Apply button. Before the click-through change, the worker reads zero fields
// because the form container is display:none; after the change it clicks Apply, waits for the
// form to render, and reads the questions.
const fixtureAshbyPage = `<!doctype html>
<html lang="en">
<head><title>Senior Backend Engineer — AcmeCorp</title></head>
<body>
  <main>
    <h1>Senior Backend Engineer</h1>
    <p>We are looking for an experienced backend engineer to join our team.</p>
    <button id="apply-btn">Apply</button>
  </main>
  <div id="application-form-container" style="display:none">
    <form id="application-form">
      <div class="application-question">
        <label class="application-label" for="q-name">Full name</label>
        <input id="q-name" name="full_name" type="text" required>
      </div>
      <div class="application-question">
        <label class="application-label" for="q-why">Why do you want this role?</label>
        <textarea id="q-why" name="why_role"></textarea>
      </div>
    </form>
  </div>
  <script>
    document.getElementById('apply-btn').addEventListener('click', function() {
      document.getElementById('application-form-container').style.display = 'block';
    });
  </script>
</body>
</html>`

// playwrightAvailable reports whether the Playwright npm package resolves from the worker's module
// location, walking up the node_modules chain as Node does.
func playwrightAvailable(root string) bool {
	for _, dir := range []string{
		filepath.Join(root, "workers", "playwright", "node_modules", "playwright"),
		filepath.Join(root, "workers", "node_modules", "playwright"),
		filepath.Join(root, "node_modules", "playwright"),
		filepath.Join(root, "..", "node_modules", "playwright"),
	} {
		if _, err := os.Stat(dir); err == nil {
			return true
		}
	}
	return false
}

func find(fields []Question, controlName string) (Question, bool) {
	for _, f := range fields {
		if f.ControlName == controlName {
			return f, true
		}
	}
	return Question{}, false
}

func hasOption(opts []string, want string) bool {
	for _, o := range opts {
		if o == want {
			return true
		}
	}
	return false
}

// TestInspectAshbyClickThrough drives the Playwright worker against the Ashby fixture page, which
// hides the application form behind an Apply button. The worker must click the Apply button and
// wait for the form to render before reading controls; without the click-through, zero fields
// are returned because the form container is display:none.
func TestInspectAshbyClickThrough(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not installed; skipping Playwright worker inspection")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	if !playwrightAvailable(root) {
		t.Skip("playwright not installed for the worker; run backend/workers/playwright/setup.sh")
	}

	w := worker.Worker{
		Command: "node",
		Args:    []string{filepath.Join(root, "workers", "playwright", "index.mjs")},
		Dir:     root,
		Timeout: 90 * time.Second,
	}
	q, err := InspectForm(context.Background(), w, Params{HTML: fixtureAshbyPage, ATSFamily: "ashby"})
	if err != nil {
		t.Fatalf("inspect Ashby fixture via worker: %v", err)
	}
	if len(q.Fields) == 0 {
		t.Fatalf("expected fields from Ashby fixture after click-through; got zero (status=%q, gaps=%v)", q.Status, q.Gaps)
	}
	if _, ok := find(q.Fields, "full_name"); !ok {
		t.Errorf("expected field full_name in Ashby fixture; fields: %+v", q.Fields)
	}
	if _, ok := find(q.Fields, "why_role"); !ok {
		t.Errorf("expected field why_role in Ashby fixture; fields: %+v", q.Fields)
	}
}

// TestInspectGenericInlineFormUnaffected verifies that a generic inline form (no ATS family) is
// still read correctly after the click-through change. The bounded wait must not interfere with
// forms already visible on the page.
func TestInspectGenericInlineFormUnaffected(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not installed; skipping Playwright worker inspection")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	if !playwrightAvailable(root) {
		t.Skip("playwright not installed for the worker; run backend/workers/playwright/setup.sh")
	}

	w := worker.Worker{
		Command: "node",
		Args:    []string{filepath.Join(root, "workers", "playwright", "index.mjs")},
		Dir:     root,
		Timeout: 90 * time.Second,
	}
	// No ATSFamily: the worker must read the inline form unchanged.
	q, err := InspectForm(context.Background(), w, Params{HTML: fixtureForm})
	if err != nil {
		t.Fatalf("inspect generic fixture via worker: %v", err)
	}
	if q.Status != "partial" {
		t.Fatalf("status: got %q want %q", q.Status, "partial")
	}
	if _, ok := find(q.Fields, "name"); !ok {
		t.Errorf("expected field name in generic fixture; fields: %+v", q.Fields)
	}
	if _, ok := find(q.Fields, "cover"); !ok {
		t.Errorf("expected field cover in generic fixture; fields: %+v", q.Fields)
	}
}

// TestInspectFormViaWorker drives the real Playwright worker over the protocol against a fixture
// form and checks the questionnaire fields come back through the Go core. The Go core contains no
// browser-automation code: it only exchanges JSON with the worker.
func TestInspectFormViaWorker(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not installed; skipping Playwright worker inspection")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	if !playwrightAvailable(root) {
		t.Skip("playwright not installed for the worker; run npm install in workers/playwright")
	}

	w := worker.Worker{
		Command: "node",
		Args:    []string{filepath.Join(root, "workers", "playwright", "index.mjs")},
		Dir:     root,
		Timeout: 90 * time.Second,
	}
	q, err := InspectForm(context.Background(), w, Params{HTML: fixtureForm})
	if err != nil {
		t.Fatalf("inspect form via worker: %v", err)
	}
	if q.Status != "partial" {
		t.Fatalf("status: got %q want %q", q.Status, "partial")
	}
	if q.Language != "en" {
		t.Fatalf("language: got %q want %q", q.Language, "en")
	}

	name, ok := find(q.Fields, "name")
	if !ok || name.Type != "text" || name.Label != "Full name" || name.MaxLength == nil || *name.MaxLength != 100 || !name.Required {
		t.Fatalf("name field wrong: %+v (ok=%v)", name, ok)
	}
	cover, ok := find(q.Fields, "cover")
	if !ok || cover.Type != "textarea" || cover.Label != "Cover letter" {
		t.Fatalf("cover field wrong: %+v (ok=%v)", cover, ok)
	}
	relocate, ok := find(q.Fields, "relocate")
	if !ok || relocate.Type != "radio" || relocate.Label != "Willing to relocate?" ||
		!hasOption(relocate.Options, "Yes") || !hasOption(relocate.Options, "No") {
		t.Fatalf("relocate field wrong: %+v (ok=%v)", relocate, ok)
	}
	cv, ok := find(q.Fields, "cv")
	if !ok || cv.Type != "file" || cv.Label != "Upload CV" {
		t.Fatalf("cv field wrong: %+v (ok=%v)", cv, ok)
	}
}
