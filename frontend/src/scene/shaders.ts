// GLSL 3 shaders for representative markers. Visibility, structural filters
// and highlights are applied per marker from its tag bitmask.

export const markerVertex = /* glsl */ `
in float tags;

uniform float uSize;        // world size, kpc
uniform float uScale;       // pixels per world unit at unit distance
uniform float uMinPx;
uniform float uMaxPx;
uniform float uVisible;     // component visibility (matter filter)
uniform float uFilterMask;  // structural filter (0 = none)
uniform float uFilterOr;    // 1: visible OR tag; 0: visible AND tag
uniform float uHighlightMask;
uniform float uSelected;    // whole component selected
uniform float uDim;         // something else is selected

out float vAlpha;
out float vHi;

void main() {
  uint t = uint(tags);
  uint fm = uint(uFilterMask);
  bool inTag = fm == 0u ? true : (t & fm) != 0u;
  bool shown;
  if (fm == 0u) {
    shown = uVisible > 0.5;
  } else if (uFilterOr > 0.5) {
    shown = uVisible > 0.5 || (t & fm) != 0u;
  } else {
    shown = uVisible > 0.5 && inTag;
  }
  if (!shown) {
    gl_Position = vec4(2.0, 2.0, 2.0, 1.0);
    gl_PointSize = 0.0;
    vAlpha = 0.0;
    vHi = 0.0;
    return;
  }
  vec4 mv = modelViewMatrix * vec4(position, 1.0);
  gl_Position = projectionMatrix * mv;
  float px = uSize * uScale / max(-mv.z, 1e-9);
  float a = 1.0;
  if (px < uMinPx) {
    // Keep sub-pixel markers' integrated brightness constant.
    a = max(px * px / (uMinPx * uMinPx), 0.02);
    px = uMinPx;
  }
  px = min(px, uMaxPx);
  uint hm = uint(uHighlightMask);
  float hi = max(uSelected, (hm != 0u && (t & hm) != 0u) ? 1.0 : 0.0);
  vHi = hi;
  vAlpha = a * (hi > 0.5 ? 1.6 : (uDim > 0.5 ? 0.3 : 1.0));
  gl_PointSize = px * (hi > 0.5 ? 1.25 : 1.0);
}
`;

export const markerFragment = /* glsl */ `
precision highp float;
uniform vec3 uColor;
uniform vec3 uHiColor;
uniform float uOpacity;
uniform float uSharp;      // 0 soft gaussian, 1 crisp disc
in float vAlpha;
in float vHi;
out vec4 fragColor;

void main() {
  vec2 c = gl_PointCoord * 2.0 - 1.0;
  float r2 = dot(c, c);
  if (r2 > 1.0) discard;
  float soft = exp(-r2 * 4.0);
  float crisp = smoothstep(1.0, 0.7, r2);
  float a = mix(soft, crisp, uSharp) * uOpacity * vAlpha;
  vec3 col = mix(uColor, uHiColor, vHi * 0.7);
  fragColor = vec4(col, clamp(a, 0.0, 1.0));
}
`;
