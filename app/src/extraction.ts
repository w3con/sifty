import { readFileSync } from 'node:fs';
import { z } from 'zod';
import { cleanHtml } from './context.js';
import { hash, providers, rootPath, type JobInput } from './core.js';
import { infer } from './providers.js';
import type { Store } from './store.js';
export const extractionSchema=z.object({title:z.string().min(1).max(400),company:z.string().nullable(),description:z.string().min(40),location:z.string().nullable()}).strict();
export async function extractWithModel(html:string,url:string,source:string,inference:typeof infer=infer,store?:Store):Promise<JobInput> {
  if(store){const {exclusive}=await import('./services.js');return exclusive(store,()=>extractWithModel(html,url,source,inference),'inference');}
  const text=cleanHtml(html),config=providers(),name=config.stages.extraction,p=config.providers[name];
  const prompt=readFileSync(rootPath('prompts/extraction.md'),'utf8');
  const data=extractionSchema.parse(await inference(p,`${prompt}\nReturn JSON matching ${JSON.stringify(z.toJSONSchema(extractionSchema))}. Every non-null field must be an EXACT contiguous quote from PAGE; description must be the available job description or an explicitly limited excerpt, not a summary. Do not include navigation.\nPAGE URL ${url}\nPAGE DATA\n${text}`,z.toJSONSchema(extractionSchema)));
  for(const [key,value] of Object.entries(data)) if(value!==null&&!text.includes(value)) throw new Error(`Model extraction ${key} is not a source quote`);
  return {title:data.title,company:data.company??'Unknown',description:data.description,descriptionStatus:`model-selected quoted excerpt; full coverage unverified; ${name}/${p.model}; prompt ${hash(prompt)}; page ${hash(text)}`,location:data.location??undefined,originalUrl:url,applyUrl:null,source};
}
