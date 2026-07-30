'use client';

import { FormEvent, useEffect, useState } from 'react';

type Settings={host:string;port:number;security:string;username:string;passwordSet:boolean;fromName:string;fromEmail:string};
const empty:Settings={host:'',port:587,security:'STARTTLS',username:'',passwordSet:false,fromName:'DETA MRP',fromEmail:''};

export function SMTPSettings(){
  const [value,setValue]=useState<Settings>(empty);const [password,setPassword]=useState('');const [testTo,setTestTo]=useState('');
  const [message,setMessage]=useState('');const [error,setError]=useState('');const [busy,setBusy]=useState(false);
  useEffect(()=>{fetch('/api/email/smtp-settings',{credentials:'include'}).then(async r=>{if(!r.ok)throw new Error();setValue(await r.json())}).catch(()=>setError('SMTP settings could not be loaded.'))},[]);
  function update<K extends keyof Settings>(key:K,v:Settings[K]){setValue(x=>({...x,[key]:v}))}
  async function save(e:FormEvent){e.preventDefault();setBusy(true);setError('');setMessage('');const r=await fetch('/api/email/smtp-settings',{method:'PUT',credentials:'include',headers:{'Content-Type':'application/json'},body:JSON.stringify({...value,password})});const p=await r.json().catch(()=>({}));setBusy(false);if(!r.ok){setError(p.message??Object.values(p.fields??{})[0]??'SMTP settings could not be saved.');return}setValue(p);setPassword('');setMessage('SMTP settings saved.')}
  async function test(){setBusy(true);setError('');setMessage('');const r=await fetch('/api/email/smtp-test',{method:'POST',credentials:'include',headers:{'Content-Type':'application/json'},body:JSON.stringify({to:testTo})});const p=await r.json().catch(()=>({}));setBusy(false);if(!r.ok){setError(p.message??'Test email could not be sent.');return}setMessage('Test email sent successfully.')}
  return <section className="settings-page"><div className="page-title-row"><div><h1>SMTP Settings</h1><p className="muted">Live email configuration for approval and supplier documents.</p></div></div>
    <form className="settings-card" onSubmit={save}><div className="crud-fields crud-fields--two">
      <label>SMTP Host<input value={value.host} onChange={e=>update('host',e.target.value)} placeholder="smtp.company.local" required/></label>
      <label>Port<input type="number" min="1" max="65535" value={value.port} onChange={e=>update('port',Number(e.target.value))} required/></label>
      <label>Security<select value={value.security} onChange={e=>update('security',e.target.value)}><option>STARTTLS</option><option>TLS</option><option>NONE</option></select></label>
      <label>Username<input value={value.username} onChange={e=>update('username',e.target.value)} autoComplete="username"/></label>
      <label>Password<input type="password" value={password} onChange={e=>setPassword(e.target.value)} placeholder={value.passwordSet?'Password saved — leave blank to keep':'SMTP password'} autoComplete="new-password"/></label>
      <label>From Name<input value={value.fromName} onChange={e=>update('fromName',e.target.value)} required/></label>
      <label>From Email<input type="email" value={value.fromEmail} onChange={e=>update('fromEmail',e.target.value)} required/></label>
    </div>{error&&<p className="form-error" role="alert">{error}</p>}{message&&<p role="status">{message}</p>}<div className="settings-actions"><button className="primary-button" disabled={busy}>{busy?'Working...':'Save settings'}</button></div></form>
    <div className="settings-card"><h2>Send Test Email</h2><div className="settings-test-row"><input type="email" value={testTo} onChange={e=>setTestTo(e.target.value)} placeholder="recipient@company.com"/><button type="button" onClick={test} disabled={busy||!testTo}>Send test email</button></div></div>
  </section>
}
