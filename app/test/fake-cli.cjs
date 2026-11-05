#!/usr/bin/env node
// Controlled integration fixture. Never contacts a real provider or executes any tool.
const path = require('node:path');
let prompt = '';
process.stdin.on('data',b=>prompt+=b);
process.stdin.on('end',()=>{
  if (process.argv.includes('status')) {console.log(JSON.stringify({loggedIn:false}));return;}
  if (prompt.includes('FCTO_TEST_TIMEOUT')) {setTimeout(()=>{},30000);return;}
  const text = prompt.includes('FCTO_TEST_MALFORMED') ? '{bad JSON' : JSON.stringify({score:7,summary:'Controlled provider fixture, not a real assessment.',evidence:[],gaps:['Fixture only']});
  if (path.basename(process.argv[1]) === 'claude') console.log(JSON.stringify({result:text,is_error:false}));
  else console.log(JSON.stringify({type:'text',part:{type:'text',text}}));
});
