export function concisePreparationError(error:unknown):string{
 const text=String(error).replace(/\u001b\[[0-?]*[ -/]*[@-~]/g,'').replace(/\\u001b\[[0-?]*[ -/]*[@-~]/g,'');
 if(/download is starting/i.test(text))return 'The page could not be opened as a questionnaire. Retry inspection for a clearer access status, or paste questions manually.';
 if(/timeout/i.test(text))return 'The application page did not respond in time. Retry inspection or add questions manually.';
 if(/browser.*(?:closed|crash)|Target.*closed/i.test(text))return 'The inspection browser closed. Close any owner-login browser and retry.';
 if(/ERR_NAME_NOT_RESOLVED|ENOTFOUND|EAI_AGAIN/i.test(text))return 'The application host could not be resolved. Check the link and network, then retry or paste questions.';
 if(/ERR_CERT|certificate/i.test(text))return 'The application site certificate could not be verified. Check the destination in your browser before retrying.';
 if(/ERR_BLOCKED|ERR_ABORTED|interrupted by another navigation/i.test(text))return 'The site interrupted or blocked inspection. Open it in the owner browser, then retry or paste questions.';
 return text.split(/\n|Call log:/)[0].replace(/^(?:(?:Error|page\.goto):\s*)+/,'').slice(0,350)||'Preparation failed. Retry or add questions manually.';
}
