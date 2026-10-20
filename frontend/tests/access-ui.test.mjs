import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';

test('private monitor gate has a bounded retry and returns to the public HTTPS portal',async()=>{
 const app=await readFile(new URL('../src/App.tsx',import.meta.url),'utf8');
 const header=await readFile(new URL('../src/components/Header.tsx',import.meta.url),'utf8');
 assert.match(app,/setTimeout\(\(\)=>controller\?\.abort\(\),8000\)/);
 assert.match(app,/signal:controller.signal/);
 assert.match(app,/generation.current\+\+;controller\?\.abort/);
 assert.match(app,/setAttempt\(value=>value\+1\)/);
 assert.match(app,/https:\/\/fribench.cn\//);
 assert.match(header,/return "https:\/\/fribench.cn\/"/);
 assert.doesNotMatch(header,/21000/);
 assert.match(app,/登录不会自动授予服务器监控权限/);
});
