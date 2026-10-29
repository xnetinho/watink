import '@testing-library/jest-dom';
import { i18n } from './translate/i18n';

// Os testes de componente rodam sempre em português, independente do idioma do navegador de quem executa (o jsdom
// detecta en-US). Testes que verificam outro idioma passam { lng } na chamada de i18n.t.
void i18n.changeLanguage('pt');
