// @vitest-environment jsdom
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { apiService } from '../src/services/api';
beforeEach(() => localStorage.clear());
afterEach(() => vi.unstubAllGlobals());
it('surfaces denied maintenance and Last.fm requests', async () => {
 vi.stubGlobal('fetch',vi.fn().mockResolvedValue(new Response('Denied',{status:403})));
 await expect(apiService.runPluginJob('deduper')).rejects.toThrow('Failed to start');
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
