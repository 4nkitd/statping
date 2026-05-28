# Design System: Statping Landing
**Project ID:** github.com/ankityadav/statping (landing site)

## 1. Visual Theme & Atmosphere
The aesthetic is **"calm control room at night."** Dark, dense, and technical — the
mood of a developer who trusts their tools. Surfaces are near-black with a faint
cool-blue undertone, so the only things that truly glow are the signals that
matter: the traffic-light trio (green / amber / red) and crisp monospace numbers.

The vibe is **precise, confident, and quietly alive.** Motion is restrained —
a single pulsing green lamp, a sparkline that breathes — never decorative noise.
Generous negative space keeps it from feeling like a dashboard dump; it reads as
a product, not a config screen. Think "GitHub dark meets a terminal you actually
enjoy looking at."

## 2. Color Palette & Roles
- **Abyss Navy-Black (#0c0e14)** — Primary page background; the deep base everything floats on.
- **Slate Panel (#14171f)** — Secondary surface for sections that need separation.
- **Raised Card (#1a1e27)** — Cards, mockups, and elevated containers.
- **Tertiary Fill (#232834)** — Inset chips, code blocks, badge backgrounds.
- **Hairline Border (#2b313c)** — 1px strokes that define edges without shouting.
- **Cloud Text (#e6e9ef)** — Primary text; high-contrast headings and body.
- **Muted Steel (#8b93a3)** — Secondary text, labels, captions.
- **Signal Green (#3fdc7f)** — The hero accent. "All systems operational," success, the lit lamp.
- **Caution Amber (#ffc24b)** — Degraded / paused / "no internet" state.
- **Alert Red (#ff5a52)** — Down state, destructive actions.
- **Electric Azure (#5aa8ff)** — Interactive accent: links, focus rings, secondary CTAs.
- **Soft Iris (#bd93f9)** — Occasional highlight for selected/featured elements.

The **green→amber→red traffic-light trio** is the brand's signature device and
appears in the logo, hero, and section accents.

## 3. Typography Rules
- **Display / Headings:** A geometric-humanist sans (Inter / system UI) at heavy
  weight (700–800), tight letter-spacing (-0.02em) for large sizes. Headlines are
  big and declarative.
- **Body:** Same sans at 400–500, 1.6 line-height, Muted-Steel-to-Cloud contrast.
- **Code & Metrics:** A monospace face (JetBrains Mono / SF Mono / ui-monospace).
  Used for terminal mockups, install commands, latency numbers, and small
  "system label" eyebrows in UPPERCASE with wide tracking (+0.12em).
- Numbers in the dashboard mockup are always monospace so columns align.

## 4. Component Stylings
* **Buttons:**
  - *Primary:* Signal-Green fill, near-black text, pill-to-soft-rounded (10px),
    subtle outer glow on hover (green bloom). Confident, "go" energy.
  - *Secondary:* Transparent with Hairline border, Cloud text; border brightens
    to Electric Azure on hover.
* **Cards / Containers:** Raised-Card background, **gently rounded corners (14–16px)**,
  1px Hairline border. Elevation is **whisper-soft** — large, low-opacity shadows
  (0 10px 40px rgba(0,0,0,0.4)) rather than hard drop shadows. Featured cards get a
  thin top accent line in the traffic-light gradient.
* **Inputs / Code blocks:** Tertiary-Fill background, Hairline stroke, monospace
  text, a small copy button on the right. Focus shows an Azure ring (3px, 15% alpha).
* **Badges / Pills:** Tertiary fill, rounded-full, tiny uppercase mono label.
  Status badges tint their background to the matching signal color at ~15% alpha.

## 5. Layout Principles
- **Centered single column, max-width ~1100px**, generous side gutters; sections
  breathe with 6–8rem vertical rhythm.
- **Asymmetry where it earns attention:** the hero pairs a left-aligned text block
  with a floating traffic-light + dashboard mockup, not a rigid 50/50.
- **Feature grid:** responsive auto-fit cards (min ~300px), equal height, even gaps.
- **Whitespace is structural** — it groups related ideas and lets the green accents
  pop. Nothing is edge-to-edge except the deep background.
- Mobile collapses to a single column; the dashboard mockup scales down but keeps
  its monospace alignment.
</content>
