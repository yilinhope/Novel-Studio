---
name: Novel Studio IDE
colors:
  surface: '#0f131c'
  surface-dim: '#0f131c'
  surface-bright: '#353942'
  surface-container-lowest: '#0a0e16'
  surface-container-low: '#181c24'
  surface-container: '#1c2028'
  surface-container-high: '#262a33'
  surface-container-highest: '#31353e'
  on-surface: '#dfe2ee'
  on-surface-variant: '#c7c4d7'
  inverse-surface: '#dfe2ee'
  inverse-on-surface: '#2c3039'
  outline: '#908fa0'
  outline-variant: '#464554'
  surface-tint: '#c0c1ff'
  primary: '#c0c1ff'
  on-primary: '#1000a9'
  primary-container: '#8083ff'
  on-primary-container: '#0d0096'
  inverse-primary: '#494bd6'
  secondary: '#4edea3'
  on-secondary: '#003824'
  secondary-container: '#00a572'
  on-secondary-container: '#00311f'
  tertiary: '#ffb95f'
  on-tertiary: '#472a00'
  tertiary-container: '#ca8100'
  on-tertiary-container: '#3e2400'
  error: '#ffb4ab'
  on-error: '#690005'
  error-container: '#93000a'
  on-error-container: '#ffdad6'
  primary-fixed: '#e1e0ff'
  primary-fixed-dim: '#c0c1ff'
  on-primary-fixed: '#07006c'
  on-primary-fixed-variant: '#2f2ebe'
  secondary-fixed: '#6ffbbe'
  secondary-fixed-dim: '#4edea3'
  on-secondary-fixed: '#002113'
  on-secondary-fixed-variant: '#005236'
  tertiary-fixed: '#ffddb8'
  tertiary-fixed-dim: '#ffb95f'
  on-tertiary-fixed: '#2a1700'
  on-tertiary-fixed-variant: '#653e00'
  background: '#0f131c'
  on-background: '#dfe2ee'
  surface-variant: '#31353e'
typography:
  display-workbench:
    fontFamily: Inter
    fontSize: 20px
    fontWeight: '600'
    lineHeight: 28px
    letterSpacing: -0.015em
  headline-panel:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: '600'
    lineHeight: 20px
    letterSpacing: -0.005em
  panel-caps:
    fontFamily: Inter
    fontSize: 11px
    fontWeight: '700'
    lineHeight: 16px
    letterSpacing: 0.05em
  ui-base:
    fontFamily: Inter
    fontSize: 13px
    fontWeight: '400'
    lineHeight: 18px
    letterSpacing: '0'
  ui-medium:
    fontFamily: Inter
    fontSize: 13px
    fontWeight: '500'
    lineHeight: 18px
    letterSpacing: '0'
  ui-micro:
    fontFamily: Inter
    fontSize: 11px
    fontWeight: '500'
    lineHeight: 14px
    letterSpacing: 0.02em
  prose-h1:
    fontFamily: Noto Serif
    fontSize: 24px
    fontWeight: '700'
    lineHeight: 36px
    letterSpacing: -0.01em
  prose-h2:
    fontFamily: Noto Serif
    fontSize: 20px
    fontWeight: '600'
    lineHeight: 30px
    letterSpacing: -0.005em
  prose-body:
    fontFamily: Noto Serif
    fontSize: 16px
    fontWeight: '400'
    lineHeight: 30px
    letterSpacing: 0.01em
  mono-code:
    fontFamily: JetBrains Mono
    fontSize: 12px
    fontWeight: '400'
    lineHeight: 18px
    letterSpacing: '0'
  mono-stats:
    fontFamily: JetBrains Mono
    fontSize: 11px
    fontWeight: '500'
    lineHeight: 14px
    letterSpacing: 0.02em
rounded:
  sm: 0.125rem
  DEFAULT: 0.25rem
  md: 0.375rem
  lg: 0.5rem
  xl: 0.75rem
  full: 9999px
spacing:
  gutter: 0.5rem
  gutter-dense: 0.25rem
  margin: 1rem
  margin-screen: 0px
  space-xs: 0.25rem
  space-sm: 0.5rem
  space-md: 0.75rem
  space-lg: 1rem
  space-xl: 1.5rem
---

## Brand & Style

This design system establishes an uncompromising, production-grade desktop workstation aesthetic for long-form narrative architects, authors, and AI orchestration pipelines. Drawing direct heritage from high-density, focused developer environments like Linear, Cursor, VS Code, and Obsidian, it rejects conversational chat bubbles, toy-like gradients, and marketing fluff in favor of an authentic instrument grade writing workbench.

The aesthetic philosophy centers on:
- **Focused Precision**: Monolithic, distraction-free obsidian and slate canvases anchored by 1px hairline boundary dividers.
- **Dual Psychology**: Strict technical density in the operational chrome juxtaposed against an expansive, serene, optical-grade serif sanctuary for prose immersion.
- **Mechanical Tactility**: High-performance micro-interactions (0.12s–0.20s easing curves, active compression, crisp badge signals) communicating continuous system state, ReAct orchestration loops, and persistent disk safety.

## Colors

The palette is engineered around a calibrated slate-graphite luminance ladder for extended writing sessions under dark conditions. High-saturation accents are strictly functional, assigned directly to agent roles and autonomous system states.

### Palette Architecture
- **Base Canvas (`#0B0F17`)**: Base frame underlay, workbench titlebar, and persistent telemetry footer.
- **Deep Slate Surface (`#111827`)**: Structural chrome, hierarchical tree panel, and right context inspector.
- **Elevated Slate (`#161F30`)**: Raised cards, popovers, tool execution headers, and modal dialogues.
- **Interactive Hover Slate (`#283548`)**: High-contrast hover feedback across lists and tree nodes.
- **Active Selection Slate (`#314158`)**: Active selections, focused tree rows, and pressed buttons.
- **Prose Chamber (`#0D131F`)**: Deep midnight blue-slate dedicated to manuscript composition, tuned to reduce optical fatigue.
- **Hairline Neutral Border (`#1F2937`)**: Crisp 1px structural separator across all panels and splits.
- **Contrast Border (`#2D3748`)**: Elevated boundary for active cards, input focus defaults, and dialog edges.

### Functional Agent & Telemetry Roles
- **Architect & AI Core (`#6366F1`)**: Primary ReAct loop execution, synopsis/beat generation, and primary interactive signals.
- **Writer & Saved State (`#10B981`)**: Manuscript drafting agent, saved disk status, and validated consistency checks.
- **Editor & Needs Review (`#F59E0B`)**: Human-in-the-loop critique flags, pending confirmations, and warning telemetry.
- **Unsynced & Error (`#EF4444`)**: Unsaved disk buffers (`未同步`), network interruption, and LLM execution failures.
- **Arbiter & Knowledge Base (`#06B6D4`)**: Vector retrieval, fact graph checks, and world lore validation.
- **Modified State (`#3B82F6`)**: Dirty tab indicators, active draft revisions, and diff modifications.

## Typography

The design system operates a strict tri-font typographic pipeline that bifurcates engineering telemetry from creative writing:

1. **System Interface Chrome (`Inter`)**: Applied universally to menus, tree navigation, buttons, inspector inputs, and modal dialogs. Tight tracking, crisp baseline alignment, and uppercase treatment on section category labels (`panel-caps`).
2. **Manuscript Sanctuary (`Noto Serif`)**: The core long-form reading and editing zone. Set at 16px with an expansive 30px line height (1.875 ratio) to allow deep cognitive focus. Center-constrained to an optimal line measure of 680px–760px.
3. **Telemetry & Execution Monospace (`JetBrains Mono`)**: Strict, tabular character alignment for prompt tokens, API cost calculation (`$0.0042`), latency readouts (`412ms`), SHA commits, and ReAct tool payload logs.

## Layout & Spacing

The layout is desktop-first, structured as a non-scrolling IDE shell (`overflow: hidden` on viewport) with compartmentalized internal scroll bays.

### Workstation Structural Geometry
- **Workbench Titlebar**: Fixed 36px height (or 40px comfortable mode) with zero margin, flush to top edge.
- **Activity Rail**: Fixed 72px width, anchoring primary workspace domains.
- **Collapsible Story Tree Sidebar**: Default 260px width (collapsible, min 200px resize bound).
- **Central Workspace**: Fluid canvas split between tab strips, active manuscripts, diff viewers, and bottom task bays.
- **Context Inspector**: Default 320px width (collapsible, min 220px resize bound).
- **Collapsible Bottom Drawer**: Default 160px height for task execution and ReAct streaming logs.
- **Persistent Status Footer**: Fixed 24px height, full-bleed across window bottom.

### Rhythmic Discipline
- Built strictly on a 4px baseline rhythm. UI paddings and element gaps default to `space-xs` (4px) or `space-sm` (8px) within panels to achieve high density.
- The manuscript chamber breaks this density deliberately: applying a minimum top/bottom padding of 48px and auto margins to maintain prose isolation.

## Elevation & Depth

Visual hierarchy is communicated through tonal layering and razor-sharp hairline borders rather than heavy blur shadows:

- **Tonal Stepping**: Depth ascends from canvas (`#0B0F17`) to panel chrome (`#111827`) to raised cards/dialogs (`#161F30`).
- **Hairline Outlines**: Elements define their spatial footprint using a 1px border of `#1F2937` or elevated `#2D3748`.
- **Minimal Ambient Elevation**: Modals and floating menus employ a structured shadow (`0 8px 24px rgba(0, 0, 0, 0.5)`) bordered by `#2D3748` over a `rgba(0, 0, 0, 0.6)` backdrop overlay.
- **Glow Accents**: Live AI agent processes emit subtle, focused chromatic halos (`0 0 12px rgba(99, 102, 241, 0.25)`), avoiding global visual noise.

## Shapes

The interface implements a compact, disciplined corner radius system (Level 1: Soft). This reinforces an architectural, mechanical desktop character while preventing visual friction at high density:

- **Micro Handles & Tree Items**: 4px radius (`rounded-sm`).
- **Standard Controls, Cards, & Tool Blocks**: 6px radius (`rounded-md`).
- **Modals & Popovers**: 8px radius (`rounded-lg`).
- **Status Badges & Pill Gauges**: Full pill (`rounded-full`, 9999px) to establish immediate shape differentiation from rectangular interactive buttons.

## Components

### Buttons & Command Triggers
- **Primary Action (`.btn-primary`)**: Solid `#6366F1` background, `#4F46E5` hover, 1px `#4F46E5` border, white text, 12px font size, 28px height, 6px radius, `active:scale-[0.97]`.
- **AI Engine Command (`variant="ai"`)**: Gradient highlight over `#6366F1`, sparkle glyph prefix, luminous violet hover accent with `shadow-[0_0_12px_rgba(99,102,241,0.3)]`.
- **Destructive Action**: Coral Red `#EF4444`, hover `#DC2626`, white text.
- **Ghost / Tool Bar**: Transparent base, `#1F2937` hover fill, `#9CA3AF` text mutating to `#F3F4F6`.
- **Micro Icon Buttons**: 22px × 22px or 28px × 28px, centered flex glyphs.

### Status Badges & Pills
Full pill shapes (`rounded-full`), 18px height, 0 7px padding, 11px font size, medium weight:
- **Saved**: Emerald tint (`rgba(16, 185, 129, 0.12)`), `#10B981` label, 5px solid emerald dot.
- **Unsynced**: Coral Red tint (`rgba(239, 68, 68, 0.15)`), `#EF4444` label (`未同步` or `Unsynced`).
- **Waiting Review**: Amber tint (`rgba(245, 158, 11, 0.15)`), `#F59E0B` label (`待审稿` or `Needs Review`).
- **Modified**: Blue tint (`rgba(59, 130, 246, 0.15)`), `#3B82F6` label.
- **Running / Agent Active**: Indigo tint (`rgba(99, 102, 241, 0.15)`), `#818CF8` label, animated 1.5s pulse.

### Hierarchical Tree Navigation
- 26px row heights, 1px 6px margins, 4px corner radius.
- Inactive rows: `#9CA3AF` text. Hover rows: `#1F2937` surface with `#F3F4F6` text.
- Active selected node: `#283548` background, `#F3F4F6` text, left 2px solid `#6366F1` indicator line.

### Inputs & Textareas
- Height: 28px (`h-7`), 6px radius, `#111827` surface, 1px `#1F2937` border, `#F3F4F6` text.
- Focus: Border shifts to `#6366F1` with a 2px outer ring `rgba(99, 102, 241, 0.3)`.

### Cards & Tool Call Execution Blocks
- **Panel Containers**: `#161F30` surface, 1px `#1F2937` boundary, 6px radius.
- **Tool Call Block**:
  - Header: 28px height, `#161F30` background, tool glyph, execution timing in `JetBrains Mono` (`142ms`), collapsible chevron.
  - Body: Deep black-slate `#080C14`, 11px monospace, syntax-colored JSON payload.

### Telemetry Statusbar
- 24px fixed height, `#0B0F17` background, 1px top border `#1F2937`.
- Divided into segmented zones separated by hairline dividers: Project breadcrumb, Word Count Target Gauge, Active Agent Step, Latency/Cost metrics, and disk persistence state.