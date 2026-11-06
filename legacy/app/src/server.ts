import express from 'express';
import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { providers, rootPath, runtime, searchConfig } from './core.js';
import { Store } from './store.js';
import { localAvailability, providerDiagnostics, providerSmoke } from './providers.js';
import { assess, collectionProgress, enrich, exclusive, importUrl, readConfig, refreshContext, saveConfig, sourceHealth, startSearch, validateAll } from './services.js';
import { AssessmentWorker, enqueueAll, enqueueJob, queueStatus, setAssessmentEnabled } from './queue.js';
import { feedbackRecords, saveFeedback } from './feedback.js';
import { localContextFingerprint } from './context.js';
import {PreparationWorker,preparationState,queuePreparation,cancelPreparation} from './applications.js';
import {addManualQuestion,updateApplication} from './application-store.js';
import {rememberAnswer,memoriesForJob} from './application-memory.js';
import {startArchiveRetention} from './archive.js';

export async function serve(store = new Store(), port = runtime().port,options:{localEvidence?:boolean;availability?:typeof localAvailability;diagnostics?:typeof providerDiagnostics;preparation?:ConstructorParameters<typeof PreparationWorker>[1];preparationGenerate?:ConstructorParameters<typeof PreparationWorker>[2]}={}) {
  validateAll(); const app = express();
  if(options.localEvidence!==false)await refreshContext(store,searchConfig().preferences,false,true);
  let evidenceKey=options.localEvidence!==false?localContextFingerprint(searchConfig().preferences):'';
  const worker=new AssessmentWorker(store,undefined,options.availability);
  const preparation=new PreparationWorker(store,options.preparation,options.preparationGenerate);
  const stopRetention=startArchiveRetention(store);worker.start();preparation.start();
  let refreshing=false;const contextTimer=setInterval(async()=>{if(refreshing||options.localEvidence===false)return;refreshing=true;try{const key=localContextFingerprint(searchConfig().preferences);if(key!==evidenceKey){await refreshContext(store,searchConfig().preferences,false,true);evidenceKey=key;enqueueAll(store);}}catch(e){console.error('Local evidence refresh:',String(e));}finally{refreshing=false;}},30000);contextTimer.unref();
  app.use((req,res,next) => {
    // Loopback bind plus Host and Origin checks block DNS-rebinding and cross-site writes.
    if (![ `127.0.0.1:${port}`,`localhost:${port}` ].includes(req.headers.host ?? '')) { res.status(403).send('Invalid local Host'); return; }
    if (req.headers.origin && ![`http://127.0.0.1:${port}`,`http://localhost:${port}`].includes(req.headers.origin)) { res.status(403).send('Invalid local Origin'); return; }
    if (req.method !== 'GET' && req.headers['content-type']?.split(';')[0] !== 'application/json') { res.status(415).send('JSON required'); return; }
    res.setHeader('X-Content-Type-Options','nosniff'); res.setHeader('Referrer-Policy','no-referrer'); next();
  });
  app.use(express.json({limit:'1mb'}));
  app.get('/api/jobs',(req,res) => res.json(store.list(String(req.query.status ?? 'all'),String(req.query.q ?? ''))));
  app.get('/api/jobs/:id',(req,res) => res.json(store.get(Number(req.params.id))));
   app.patch('/api/jobs/:id',(req,res) => res.json(store.status(Number(req.params.id),req.body.stage??req.body.status,req.body.confirmed===true)));
   app.delete('/api/jobs/:id',(req,res)=>{if(req.body.confirmed!==true)throw Error('Permanent deletion requires explicit confirmation');res.json(store.deleteJob(Number(req.params.id)));});
  app.post('/api/jobs/:id/seen',(req,res)=>res.json(store.seen(Number(req.params.id))));
  app.get('/api/health',(_req,res) => res.json(sourceHealth(store)));
  app.get('/api/context',(_req,res) => res.json(store.context()));
  app.get('/api/filter-options',(_req,res)=>res.json(searchConfig()));
  app.get('/api/assessment-options',(_req,res)=>{const c=providers();res.json({selected:c.stages.assessment,providers:Object.entries(c.providers).map(([name,p])=>({name,model:p.model,type:p.type}))});});
  app.get('/api/stage-providers',(_req,res)=>{const c=providers();res.json({stages:{assessment:c.stages.assessment,preparation:c.stages.preparation,extraction:c.stages.extraction},providers:Object.entries(c.providers).map(([name,p])=>({name,model:p.model}))});});
  app.get('/api/queue',(_req,res)=>res.json(queueStatus(store)));
  app.get('/api/jobs/:id/application',(req,res)=>res.json(preparationState(store,Number(req.params.id))));
   app.post('/api/jobs/:id/application/prepare',(req,res)=>res.status(202).json(queuePreparation(store,Number(req.params.id),req.body)));
   app.post('/api/jobs/:id/application/cancel',(req,res)=>res.json(cancelPreparation(store,Number(req.params.id))));
  app.patch('/api/jobs/:id/application',(req,res)=>res.json(updateApplication(store,Number(req.params.id),req.body)));
  app.post('/api/jobs/:id/application/questions',(req,res)=>res.json(addManualQuestion(store,Number(req.params.id),req.body.label,req.body.type,req.body.options)));
  app.post('/api/jobs/:id/application/remember',(req,res)=>res.json(rememberAnswer(store,Number(req.params.id),req.body)));
  app.get('/api/jobs/:id/application/memories',(req,res)=>res.json(memoriesForJob(store,Number(req.params.id))));
  app.get('/api/application-files/:name',(req,res)=>{if(!['cv.pdf','profile.pdf'].includes(String(req.params.name)))throw Error('Unknown candidate file');res.setHeader('Content-Type','application/pdf');res.setHeader('Content-Disposition',`inline; filename="${req.params.name}"`);res.sendFile(rootPath(String(req.params.name)),{dotfiles:'allow'});});
  app.put('/api/queue/assessment',(req,res)=>res.json(setAssessmentEnabled(store,req.body?.enabled)));
  app.get('/api/jobs/:id/feedback',(req,res)=>res.json(feedbackRecords().filter(r=>store.resolveId(r.jobId)===store.resolveId(Number(req.params.id)))));
  app.post('/api/jobs/:id/feedback',(req,res)=>{const record=saveFeedback(store,Number(req.params.id),String(req.body.text??''),req.body.scope,req.body.supersedes);enqueueJob(store,Number(req.params.id),100,true);res.status(202).json({record,queue:queueStatus(store)});setImmediate(()=>{try{enqueueAll(store);}catch(e){console.error('Feedback reassessment scheduling:',String(e));}});});
  app.get('/api/providers',async (_req,res) => res.json(await (options.diagnostics??providerDiagnostics)()));
  app.post('/api/providers/:name/check',async (req,res) => res.json(await exclusive(store,() => providerSmoke(String(req.params.name)),'inference')));
  app.post('/api/context',async (req,res) => res.json(await exclusive(store,() => refreshContext(store,searchConfig().preferences,req.body.refreshRemote === true))));
  app.get('/api/collection',(_req,res)=>res.json(collectionProgress(store)));
  app.post('/api/search',(req,res) => res.status(202).json(startSearch(store,req.body.source,req.body.force===true)));
  app.post('/api/import',async (req,res) => res.json({id:await exclusive(store,() => importUrl(store,String(req.body.url)))}));
  app.post('/api/jobs/:id/read',async (req,res) => res.json({id:await exclusive(store,() => enrich(store,Number(req.params.id)))}));
  app.post('/api/jobs/:id/assess',(req,res) => res.status(202).json(enqueueJob(store,Number(req.params.id),100,true,req.body.provider)));
  app.get('/api/config/:name',(req,res) => res.json({content:readConfig(String(req.params.name))}));
  app.put('/api/config/:name',async (req,res) => res.json(await exclusive(store,async () => saveConfig(String(req.params.name),String(req.body.content)))));
   app.use('/api',(_req,res)=>res.status(404).json({error:'Unknown API endpoint. Refresh the page to load the current catalogue version.'}));
   const dist = rootPath('frontend/dist');
  if (!existsSync(resolve(dist,'index.html'))) throw new Error('Build UI first: npm --prefix frontend run build');
  app.use(express.static(dist));
  app.use((err: Error,_req: express.Request,res: express.Response,_next: express.NextFunction) => { console.error(err.message); res.status(400).json({error:err.message}); });
  const server = app.listen(port,'127.0.0.1',() => console.error(`Catalogue: http://127.0.0.1:${port}`));
  server.on('close',()=>{worker.stop();preparation.stop();clearInterval(contextTimer);stopRetention();});
  return server;
}
