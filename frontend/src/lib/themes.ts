// ─── Theme System ──────────────────────────────────────────────
// Each theme defines all CSS custom-property values used across the UI.
// The active theme is applied by writing these vars to :root.
// To add a new theme just add another entry to `themes`.

export interface ThemeDefinition {
  id: string;
  name: string;
  description: string;
  colors: {
    // Backgrounds
    bgPrimary: string;
    bgSecondary: string;
    bgTertiary: string;
    bgElevated: string;

    // Text
    textPrimary: string;
    textSecondary: string;
    textMuted: string;

    // Accent
    accent: string;
    accentHover: string;
    accentMuted: string; // accent at ~15% opacity feel
    accentGlow: string; // for box-shadow / glow effects

    // Borders
    border: string;
    borderSubtle: string;

    // Semantic
    success: string;
    successMuted: string;
    warning: string;
    warningMuted: string;
    error: string;
    errorMuted: string;
    info: string;
    infoMuted: string;
  };
}

// ─── Forge Theme (Default) ─────────────────────────────────────
// Warm charcoal backgrounds, ember/amber accents — a blacksmith's forge.
export const forgeTheme: ThemeDefinition = {
  id: 'forge',
  name: 'Forge',
  description: 'Warm charcoal with ember accents — the default ByteSmith experience',
  colors: {
    bgPrimary: '#0c0b0a',
    bgSecondary: '#141210',
    bgTertiary: '#1c1916',
    bgElevated: '#231f1b',

    textPrimary: '#e8e4df',
    textSecondary: '#9a9590',
    textMuted: '#6b6560',

    accent: '#e07a2f',
    accentHover: '#f0943e',
    accentMuted: '#e07a2f26',
    accentGlow: '#e07a2f40',

    border: '#2a2521',
    borderSubtle: '#1f1c19',

    success: '#4ade80',
    successMuted: '#4ade8020',
    warning: '#fbbf24',
    warningMuted: '#fbbf2420',
    error: '#f87171',
    errorMuted: '#f8717120',
    info: '#60a5fa',
    infoMuted: '#60a5fa20',
  },
};

// ─── Midnight Theme ────────────────────────────────────────────
// Cool blue-tinted dark — the original ByteSmith look.
export const midnightTheme: ThemeDefinition = {
  id: 'midnight',
  name: 'Midnight',
  description: 'Cool indigo-tinted dark theme',
  colors: {
    bgPrimary: '#0f1117',
    bgSecondary: '#1a1d27',
    bgTertiary: '#242833',
    bgElevated: '#2c3040',

    textPrimary: '#e4e4e7',
    textSecondary: '#a1a1aa',
    textMuted: '#71717a',

    accent: '#6366f1',
    accentHover: '#818cf8',
    accentMuted: '#6366f126',
    accentGlow: '#6366f140',

    border: '#2e3345',
    borderSubtle: '#232838',

    success: '#22c55e',
    successMuted: '#22c55e20',
    warning: '#f59e0b',
    warningMuted: '#f59e0b20',
    error: '#ef4444',
    errorMuted: '#ef444420',
    info: '#3b82f6',
    infoMuted: '#3b82f620',
  },
};

// ─── Obsidian Theme ────────────────────────────────────────────
// Near-black with green accents — hacker aesthetic.
export const obsidianTheme: ThemeDefinition = {
  id: 'obsidian',
  name: 'Obsidian',
  description: 'Near-black with emerald accents',
  colors: {
    bgPrimary: '#080808',
    bgSecondary: '#111111',
    bgTertiary: '#191919',
    bgElevated: '#212121',

    textPrimary: '#d4d4d4',
    textSecondary: '#888888',
    textMuted: '#555555',

    accent: '#10b981',
    accentHover: '#34d399',
    accentMuted: '#10b98126',
    accentGlow: '#10b98140',

    border: '#222222',
    borderSubtle: '#1a1a1a',

    success: '#10b981',
    successMuted: '#10b98120',
    warning: '#f59e0b',
    warningMuted: '#f59e0b20',
    error: '#ef4444',
    errorMuted: '#ef444420',
    info: '#06b6d4',
    infoMuted: '#06b6d420',
  },
};

// ─── Theme Registry ────────────────────────────────────────────

export const themes: Record<string, ThemeDefinition> = {
  forge: forgeTheme,
  midnight: midnightTheme,
  obsidian: obsidianTheme,
};

export const defaultThemeId = 'forge';

// ─── Apply theme to DOM ────────────────────────────────────────

export function applyTheme(theme: ThemeDefinition): void {
  const root = document.documentElement;
  const c = theme.colors;

  root.style.setProperty('--bg-primary', c.bgPrimary);
  root.style.setProperty('--bg-secondary', c.bgSecondary);
  root.style.setProperty('--bg-tertiary', c.bgTertiary);
  root.style.setProperty('--bg-elevated', c.bgElevated);

  root.style.setProperty('--text-primary', c.textPrimary);
  root.style.setProperty('--text-secondary', c.textSecondary);
  root.style.setProperty('--text-muted', c.textMuted);

  root.style.setProperty('--accent', c.accent);
  root.style.setProperty('--accent-hover', c.accentHover);
  root.style.setProperty('--accent-muted', c.accentMuted);
  root.style.setProperty('--accent-glow', c.accentGlow);

  root.style.setProperty('--border', c.border);
  root.style.setProperty('--border-subtle', c.borderSubtle);

  root.style.setProperty('--success', c.success);
  root.style.setProperty('--success-muted', c.successMuted);
  root.style.setProperty('--warning', c.warning);
  root.style.setProperty('--warning-muted', c.warningMuted);
  root.style.setProperty('--error', c.error);
  root.style.setProperty('--error-muted', c.errorMuted);
  root.style.setProperty('--info', c.info);
  root.style.setProperty('--info-muted', c.infoMuted);
}
