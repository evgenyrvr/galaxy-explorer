import { createRoot } from 'react-dom/client';
import App from './App';
import { LogFrontendError } from '../wailsjs/go/main/App';
import './styles.css';

// Surface uncaught errors in the window (no devtools in release builds).
function showError(msg: string) {
  LogFrontendError(msg).catch(() => {});
  const el = document.createElement('div');
  el.className = 'toast error';
  el.textContent = msg;
  el.onclick = () => el.remove();
  document.body.appendChild(el);
}
window.addEventListener('error', (e) => showError(`Error: ${e.message} ${e.error?.stack ?? ''}`));
window.addEventListener('unhandledrejection', (e) => showError(`Error: ${String(e.reason?.message ?? e.reason)} ${e.reason?.stack ?? ''}`));

createRoot(document.getElementById('root')!).render(<App />);
