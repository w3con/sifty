// Frontend-owned types extracted from legacy/app/src/application-store.ts.
// Only the types used by the frontend are included; store.ts and its
// backend-only code are not pulled in.
import {z} from 'zod';
export const questionSchema=z.object({id:z.string(),label:z.string().min(1).max(8000),instructions:z.string().max(12000).optional(),labelStatus:z.enum(['resolved','uncertain']).optional(),type:z.enum(['text','textarea','select','radio','checkbox','file']),options:z.array(z.string()).default([]),required:z.boolean().default(false),maxLength:z.number().nullable().default(null),accept:z.string().default(''),sourceUrl:z.string(),origin:z.enum(['site','manual']),language:z.string().default('und'),signature:z.string().default(''),controlName:z.string().default('')});
export type Question=z.infer<typeof questionSchema>;
export type Questionnaire={url:string;capturedAt:string;language:string;status:'partial'|'blocked'|'login-required'|'unsupported';gaps:string[];fields:Question[];family:string;technicalLog?:string};
export type AnswerField=Question&{value:string|string[];revision:number;edited:boolean;unknowns:string[];evidence:{sourceId:string;quote:string}[];changed?:boolean;removed?:boolean};
export type ApplicationDraft={jobId:number;revision:number;fields:AnswerField[];selectedFile:'cv.pdf'|'profile.pdf';questionnaire:Questionnaire|null;provenance:any;notes:string[]};
