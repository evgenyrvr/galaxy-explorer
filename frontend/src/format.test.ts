import { describe, expect, it } from 'vitest';
import { length, parseMass, pct, sci } from './format';

describe('format', () => {
  it('writes masses in scientific notation', () => {
    expect(sci(3e10)).toBe('3.00×10¹⁰');
    expect(sci(1.4e8)).toBe('1.40×10⁸');
    expect(sci(0)).toBe('0');
  });

  it('keeps tiny percentages visible (black hole < 0.01%)', () => {
    expect(pct(1.4e-4)).toBe('0.014%');
    expect(pct(1e-6)).toBe('0.00010%');
    expect(pct(0.853246)).toBe('85.32%');
    expect(pct(0)).toBe('0%');
  });

  it('parses user mass input', () => {
    expect(parseMass('1e12')).toBe(1e12);
    expect(parseMass('2×10^12')).toBe(2e12);
    expect(parseMass('5.4e7 M☉')).toBe(5.4e7);
    expect(Number.isNaN(parseMass('abc'))).toBe(true);
  });

  it('chooses length units', () => {
    expect(length(211)).toBe('211 kpc');
    expect(length(0.004)).toBe('4.00 pc');
    expect(length(1500)).toBe('1.50 Mpc');
  });
});
