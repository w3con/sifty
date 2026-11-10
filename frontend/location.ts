// Turns a listing's location string into one readable place: a country when the
// listing names one, a region (the same regions as the profile) when it lists
// many countries, "Worldwide" when it lists most of the world. The original
// string is never changed and can always be shown in full.
import {REGION_LABELS} from './profile/referenceData';

const REGION_CODES:Record<string,string>={
  europe:'AD AL AT AX BA BE BG BY CH CY CZ DE DK EE ES FI FO FR GB GG GI GR HR HU IE IM IS IT JE LI LT LU LV MC MD ME MK MT NL NO PL PT RO RS RU SE SI SJ SK SM UA VA XK',
  asia:'AF AM AZ BD BT CN GE HK IN KG KP KR KZ LK MN MO MV NP PK TJ TM TW UZ',
  southeast_asia:'BN ID KH LA MM MY PH SG TH TL VN',
  japan_oceania:'JP AU NZ FJ PG SB VU NC PF WS TO TV KI NR PW FM MH CK NU NF TK WF GU MP AS UM PN',
  north_america:'US CA MX GL BM PM BZ CR SV GT HN NI PA AG AI AW BB BL BQ BS CU CW DM DO GD GP HT JM KN KY LC MF MQ MS PR SX TC TT VC VG VI',
  south_america:'AR BO BR CL CO EC FK GF GY PE PY SR UY VE GS',
  middle_east_africa:'AE BH EG IL IQ IR JO KW LB OM PS QA SA SY TR YE AO BF BI BJ BW CD CF CG CI CM CV DJ DZ EH ER ET GA GH GM GN GQ GW KE KM LR LS LY MA MG ML MR MU MW MZ NA NE NG RE RW SC SD SH SL SN SO SS ST SZ TD TG TN TZ UG YT ZA ZM ZW',
};
const REGION_OF=new Map<string,string>();
for(const [region,codes] of Object.entries(REGION_CODES))for(const c of codes.split(' '))REGION_OF.set(c,region);
const US_STATES=new Set('AL AK AZ AR CA CO CT DE DC FL GA HI ID IL IN IA KS KY LA ME MD MA MI MN MS MO MT NE NV NH NJ NM NY NC ND OH OK OR PA RI SC SD TN TX UT VT VA WA WV WI WY'.split(' '));
const US_STATE_NAME=/^(alabama|alaska|arizona|arkansas|california|colorado|connecticut|delaware|florida|georgia|hawaii|idaho|illinois|indiana|iowa|kansas|kentucky|louisiana|maine|maryland|massachusetts|michigan|minnesota|mississippi|missouri|montana|nebraska|nevada|new hampshire|new jersey|new mexico|new york|north carolina|north dakota|ohio|oklahoma|oregon|pennsylvania|rhode island|south carolina|south dakota|tennessee|texas|utah|vermont|virginia|washington|west virginia|wisconsin|wyoming|district of columbia)$/i;
const REMOTE=/^(telecommute|remote|work from home|wfh|fully remote)$/i;
const WORLD=/^(anywhere|worldwide|global|world)$/i;

let names:Intl.DisplayNames|null=null;
const countryName=(code:string)=>{try{names??=new Intl.DisplayNames(['en'],{type:'region'});return names.of(code)??code;}catch{return code;}};
let byName:Map<string,string>|null=null;
const codeForName=(name:string)=>{
  if(!byName){byName=new Map();for(const c of REGION_OF.keys())byName.set(countryName(c).toLowerCase(),c);
    for(const [n,c] of [['usa','US'],['united states of america','US'],['uk','GB'],['great britain','GB'],['england','GB'],['the netherlands','NL'],['south korea','KR'],['korea','KR'],['uae','AE']] as const)byName.set(n,c);}
  return byName.get(name.toLowerCase());
};

export type LocationSummary={text:string;remote:boolean;compressed:boolean;countries:number};

export function summarizeLocation(raw?:string|null):LocationSummary{
  const tokens=(raw??'').split(/\s*[·,;|/]\s*/).map(t=>t.trim()).filter(Boolean);
  const remote=tokens.some(t=>REMOTE.test(t));
  const world=tokens.some(t=>WORLD.test(t));
  const places=tokens.filter(t=>!REMOTE.test(t)&&!WORLD.test(t));
  const out=(s:string,compressed:boolean,countries=0)=>({text:[remote?'Remote':'',s].filter(Boolean).join(' · '),remote,compressed,countries});
  if(!places.length)return out(world?'Worldwide':'',false);
  const twoLetter=places.filter(t=>/^[A-Z]{2}$/.test(t)&&t!=='ZZ');
  // A list of two-letter codes is US states when most of them are not country codes.
  const statesList=twoLetter.length>=4&&twoLetter.filter(c=>US_STATES.has(c)&&!REGION_OF.has(c)).length>=twoLetter.filter(c=>REGION_OF.has(c)&&!US_STATES.has(c)).length;
  const countries=new Set<string>(),states:string[]=[],other:string[]=[];
  for(const t of places){
    if(t==='ZZ')continue;
    if(US_STATE_NAME.test(t)||(statesList&&US_STATES.has(t))){countries.add('US');states.push(t);continue;}
    const code=/^[A-Z]{2}$/.test(t)&&REGION_OF.has(t)?t:codeForName(t);
    if(code)countries.add(code);else other.push(t);
  }
  // Cities or anything not recognised: keep short strings as written.
  if(other.length&&places.length<=3)return out(places.join(' · '),false);
  if(!countries.size)return places.length<=3?out(places.join(' · '),false):out(`${places.slice(0,2).join(' · ')} +${places.length-2}`,true);
  const compressed=places.length>3;
  if(countries.size===1){const [c]=[...countries];const name=countryName(c);
    if(states.length===1)return out(`${states[0]}, ${name}`,false,1);
    return out(name,compressed,1);}
  const regions=new Set([...countries].map(c=>REGION_OF.get(c)!));
  if(countries.size>=100||regions.size>=5)return out('Worldwide',true,countries.size);
  if(countries.size<=3)return out([...countries].map(countryName).join(' · '),compressed,countries.size);
  const label=Object.keys(REGION_LABELS).filter(r=>regions.has(r)).map(r=>REGION_LABELS[r]).join(' · ');
  return out(`${label} · ${countries.size} countries`,true,countries.size);
}
