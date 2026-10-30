import { createRoot } from '@react-facet/dom-fiber';
import './app.css';
import { App } from './App';
import { startDoc } from './runtime/doc';

const host = document.createElement('div');
host.id = 'oui-ddui';
document.body.appendChild(host);
createRoot(host).render(<App />);
startDoc();
