// @vitest-environment jsdom
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { apiService } from '../src/services/api';
beforeEach(() => localStorage.clear());
afterEach(() => vi.unstubAllGlobals());
it('surfaces denied maintenance and Last.fm requests', async () => {
 vi.stubGlobal('fetch',vi.fn().mockResolvedValue(new Response('Denied',{status:403})));
 await expect(apiService.runPluginJob('deduper')).rejects.toThrow('Denied');
 await expect(apiService.getLastFmAuthUrl('https://test')).rejects.toThrow('authorization');
 await expect(apiService.exchangeLastFmToken('token')).rejects.toThrow('exchange');
});
it('keeps an account signed in when a previous account request fails', async () => {
 localStorage.setItem('sn_token','old');
 let resolve!: (response:Response)=>void;
 vi.stubGlobal('fetch',vi.fn(()=>new Promise<Response>(done=>{resolve=done})));
 const pending=apiService.fetchHearts();
 localStorage.setItem('sn_token','new'); resolve(new Response('',{status:401}));
 await expect(pending).rejects.toThrow(); expect(localStorage.getItem('sn_token')).toBe('new');
});
it('leaves multipart boundaries to the browser when importing OPML',async()=>{
 const fetch=vi.fn().mockResolvedValue(new Response('',{status:200})); vi.stubGlobal('fetch',fetch);
 localStorage.setItem('sn_token','token');
 await apiService.importOPML(new File(['<opml/>'],'feeds.opml'));
 const options=fetch.mock.calls[0][1];
 expect(options.headers.has('Content-Type')).toBe(false);
 expect(options.headers.get('Authorization')).toBe('Bearer token');
 expect(options.body).toBeInstanceOf(FormData);
});
it('evicts old lyrics and does not cache oversized responses',async()=>{
 const fetch=vi.fn().mockImplementation(()=>Promise.resolve(new Response(JSON.stringify({plainLyrics:'words'}))));vi.stubGlobal('fetch',fetch);
 for(let i=0;i<140;i++) await apiService.getLyrics('cache-test',String(i),'album',120);
 const calls=fetch.mock.calls.length;
 await apiService.getLyrics('cache-test','139','album',120);
 expect(fetch).toHaveBeenCalledTimes(calls);
 await apiService.getLyrics('cache-test','0','album',120);
 expect(fetch).toHaveBeenCalledTimes(calls+1);
 fetch.mockImplementation(()=>Promise.resolve(new Response(JSON.stringify({plainLyrics:'x'.repeat(128*1024)}))));
 await apiService.getLyrics('large','song','album',120);
 await apiService.getLyrics('large','song','album',120);
 expect(fetch).toHaveBeenCalledTimes(calls+3);
});
it('uses distinct lyrics keys even when names contain separators',async()=>{
 const fetch=vi.fn().mockImplementation(()=>Promise.resolve(new Response('{}')));vi.stubGlobal('fetch',fetch);
 await apiService.getLyrics('a-b','c','d',120);
 await apiService.getLyrics('a','b-c','d',120);
 expect(fetch).toHaveBeenCalledTimes(2);
});
it('cancels upstream lyrics work and preserves the abort reason',async()=>{
 const controller=new AbortController();
 const fetch=vi.fn((_url: string, options: RequestInit)=>new Promise<Response>((_resolve,reject)=>{
  options.signal!.addEventListener('abort',()=>reject(options.signal!.reason),{once:true});
 }));
 vi.stubGlobal('fetch',fetch);
 const pending=apiService.getLyrics('cancel-test','song','album',120,controller.signal);
 controller.abort();
 await expect(pending).rejects.toMatchObject({name:'AbortError'});
 expect(fetch.mock.calls[0][1].signal?.aborted).toBe(true);
});
