import {initInputCharCounter} from './common-page.ts';
import {createElementFromHTML} from '../utils/dom.ts';

function makeField(maxLength: number, value: string = ''): HTMLElement {
  const container = createElementFromHTML<HTMLElement>(`
    <div data-locale-chars-left-1="%d character left" data-locale-chars-left-n="%d characters left">
      <input maxlength="${maxLength}">
    </div>
  `);
  container.querySelector('input')!.value = value;
  initInputCharCounter(container);
  return container;
}

function counterOf(container: HTMLElement): HTMLElement {
  return container.querySelector('.input-char-counter')!;
}

describe('initInputCharCounter', () => {
  beforeEach(() => {
    document.documentElement.lang = 'en-US'; // trN resolves plural rules from the document language
  });

  test('uses the plural form for an empty field', () => {
    expect(counterOf(makeField(255)).textContent).toBe('255 characters left');
  });

  test('uses the singular form at one remaining', () => {
    expect(counterOf(makeField(255, 'x'.repeat(254))).textContent).toBe('1 character left');
  });

  test('recounts on input, which covers pasting', () => {
    const container = makeField(255);
    const input = container.querySelector('input')!;
    input.value = 'hello';
    input.dispatchEvent(new Event('input'));
    expect(counterOf(container).textContent).toBe('250 characters left');
  });

  test('is near-limit but not at-limit while characters remain', () => {
    const counter = counterOf(makeField(255, 'x'.repeat(235)));
    expect(counter.classList.contains('near-limit')).toBe(true);
    expect(counter.classList.contains('at-limit')).toBe(false);
  });

  test('is at-limit and not near-limit at zero', () => {
    const counter = counterOf(makeField(255, 'x'.repeat(255)));
    expect(counter.classList.contains('at-limit')).toBe(true);
    expect(counter.classList.contains('near-limit')).toBe(false);
  });

  test('reports a server-rendered value longer than maxlength as over', () => {
    const counter = counterOf(makeField(255, 'x'.repeat(258)));
    expect(counter.textContent).toBe('-3 characters left');
    expect(counter.classList.contains('at-limit')).toBe(true);
  });

  test('counts an astral character as two, matching maxlength', () => {
    expect(counterOf(makeField(255, '😀')).textContent).toBe('253 characters left');
  });

  test('does nothing when no input in the container has maxlength', () => {
    const container = createElementFromHTML<HTMLElement>('<div><input></div>');
    initInputCharCounter(container);
    expect(container.querySelector('.input-char-counter')).toBeNull();
  });
});
