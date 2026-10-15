// Number formatting with physical units.

const SUP: Record<string, string> = {
  '-': '⁻', '0': '⁰', '1': '¹', '2': '²', '3': '³', '4': '⁴', '5': '⁵', '6': '⁶', '7': '⁷', '8': '⁸', '9': '⁹',
};

export function sup(n: number | string): string {
  return String(n).split('').map((c) => SUP[c] ?? c).join('');
}

/** Scientific notation such as 3.00×10¹⁰. */
export function sci(x: number, digits = 3): string {
  if (!isFinite(x)) return '—';
  if (x === 0) return '0';
  const e = Math.floor(Math.log10(Math.abs(x)));
  if (e >= -2 && e < 4) return x.toPrecision(digits);
  const m = x / Math.pow(10, e);
  return `${m.toFixed(Math.max(0, digits - 1))}×10${sup(e)}`;
}

export function mass(x: number): string {
  if (x === 0) return '0 M☉';
  return `${sci(x)} M☉`;
}

/** Percent with enough precision for tiny shares (e.g. black hole <0.01%). */
export function pct(f: number): string {
  const p = f * 100;
  if (p === 0) return '0%';
  if (Math.abs(p) >= 10) return `${p.toFixed(2)}%`;
  if (Math.abs(p) >= 0.01) return `${p.toFixed(3)}%`;
  if (Math.abs(p) >= 1e-6) return `${p.toPrecision(2)}%`;
  return `<10${sup(-6)}%`;
}

export function length(kpc: number): string {
  const a = Math.abs(kpc);
  if (a >= 1000) return `${(kpc / 1000).toPrecision(3)} Mpc`;
  if (a >= 1) return `${kpc.toPrecision(3)} kpc`;
  if (a >= 1e-3) return `${(kpc * 1000).toPrecision(3)} pc`;
  return `${(kpc * 1e6).toPrecision(3)} mpc`;
}

/** Parses "1e12", "2×10^12", "2*10^12", "5.4e7". */
export function parseMass(s: string): number {
  const t = s.replace(/\s|M☉|Msun/gi, '').replace('×', '*').replace('x10', '*10');
  const m = t.match(/^([0-9.]+)\*10\^?([-+]?\d+)$/);
  if (m) return parseFloat(m[1]) * Math.pow(10, parseInt(m[2], 10));
  return Number(t);
}

export const EVIDENCE_LABEL: Record<string, string> = {
  OBSERVED: 'Observed',
  INFERRED: 'Inferred',
  RECONSTRUCTED: 'Reconstructed',
  ILLUSTRATIVE: 'Illustrative',
};
