// Every test entry point imports this before constructing runtime services.
// An explicit test state may be reused, but operational .local is never permitted.
const requested=process.env.FCTO_STATE_DIR;
if(requested&&(!requested.startsWith('.local/tests/')&&!requested.startsWith('.local/research/')))throw new Error('Tests require .local/tests/ or .local/research/ state; operational storage refused');
process.env.FCTO_STATE_DIR=requested??`.local/tests/${process.pid}-${Date.now()}`;
