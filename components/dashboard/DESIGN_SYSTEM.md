# Language Operator Design System

## Design Philosophy

This design system is inspired by **Donald Judd's minimalism** and the **West Texas landscape** of Marfa. It emphasizes:

- Pure geometric precision
- Restrained earth-tone palette
- Maximum negative space
- Typography as sculpture
- Material honesty (no decoration)
- Warmth revealed through interaction

---

## Color Palette

### Light Mode (West Texas Day)

#### Neutrals (Stone)
```css
stone-50:  #fafaf9   /* Subtle warm off-white */
stone-100: #f5f5f4   /* Background base */
stone-200: #e7e5e4   /* Input borders */
stone-600: #57534e   /* Secondary text */
stone-800: #292524   /* Primary dark */
stone-900: #1c1917   /* Deep charcoal */
stone-950: #0c0a09   /* Nearly black */
```

#### Warm Accents (Amber/Desert)
```css
amber-50:   #fffbeb   /* Atmospheric mist */
amber-900:  #78350f   /* Deep rust */
amber-950:  #451a03   /* Weathered bronze */
```

#### Light Mode Gradients

**Background (Atmospheric)**
```css
bg-gradient-to-br from-stone-100 via-amber-50/30 to-neutral-100
```
- Subtle warm gradient
- Barely perceptible shift from stone to amber mist
- Creates environmental warmth without visual noise

**Button (Default State)**
```css
bg-gradient-to-r from-stone-800 to-stone-950
```
- Restrained charcoal to near-black
- Warmth hidden until interaction

**Button (Hover State)**
```css
hover:from-amber-900 hover:to-amber-950
```
- Reveals desert warmth on interaction
- Rust to deep bronze

### Dark Mode (West Texas Night)

Inspired by the vast starlit nights of the Chihuahuan Desert — deep sky darkness, sage moonlight, and firelight warmth.

#### Night Neutrals
```css
neutral-950:  #0a0a0a   /* Deep desert black */
stone-900:    #1c1917   /* Warm darkness */
stone-800:    #292524   /* Darker surface */
stone-700:    #44403c   /* Medium surface */
stone-600:    #57534e   /* Divider lines */
stone-400:    #a8a29e   /* Sage moonlight (labels) */
stone-300:    #d6d3d1   /* Bright moonlight (text) */
```

#### Night Fire (Warm Interactions)
```css
amber-600:    #d97706   /* Firelight glow */
amber-400:    #fbbf24   /* Starlight (cursor) */
orange-600:   #ea580c   /* Fire embers */
```

#### Dark Mode Gradients

**Background (Night Sky)**
```css
dark:from-neutral-950 dark:via-stone-900/50 dark:to-stone-950
```
- Deep desert night with warm stone undertones
- Vast darkness with subtle horizon warmth
- Atmospheric depth without harshness

**Button (Default State)**
```css
dark:from-stone-700 dark:to-stone-800
```
- Muted stone in darkness
- Visible but restrained

**Button (Hover - Firelight)**
```css
dark:hover:from-amber-600 dark:hover:to-orange-600
```
- Reveals fire warmth on interaction
- Like lighting a match in the desert night

---

## Typography

### Font Family
```css
font-family: system-ui, -apple-system, sans-serif
font-weight: 300 /* Light weight only */
```

### Scale & Tracking

**Header (Logo)**
```css
font-size: 13px
letter-spacing: 0.2em  /* 200% */
text-transform: uppercase
```

**Labels**
```css
font-size: 10px
letter-spacing: 0.2em  /* 200% */
text-transform: uppercase
```

**Buttons**
```css
font-size: 11px
letter-spacing: 0.15em  /* 150% */
text-transform: uppercase
```

**Body/Input**
```css
font-size: 14px
font-weight: 300
```

### Principles
- Extended tracking creates sculptural quality
- All-caps for headers and UI chrome
- Light weight (300) for elegance
- Typography functions as architectural element

---

## Spacing

### Base Unit: 12px (0.75rem)

**Generous Padding**
```css
p-12  /* 48px - Primary container padding */
space-y-6  /* 24px - Form field spacing */
space-y-2  /* 8px - Label to input spacing */
gap-1  /* 4px - Cursor gap */
```

### Principles
- Embrace negative space (48px padding is deliberate)
- Vertical rhythm: 24px between major elements
- Minimal spacing only where necessary
- Space is a design material, not absence

---

## Components

### Card/Container

**Structure**
```css
max-width: 480px  /* Precise geometric constraint */
background: white/95
backdrop-filter: blur(sm)
border: 1px solid stone-800/90
box-shadow: 0 8px 32px rgba(120,53,15,0.08)  /* Warm shadow */
```

**Sections**
```css
/* Header */
border-bottom: 1px solid stone-800/80
padding: 48px

/* Content */
padding: 48px

/* Footer */
border-top: 1px solid stone-800/80
padding: 48px
```

### Form Inputs

```css
width: 100%
height: 48px
padding: 0 16px
background: stone-50/30  /* Subtle tint */
border: 1px solid stone-200
font-size: 14px
font-weight: 300

/* Focus State */
border-color: amber-900/40
ring: 1px amber-900/20
outline: none
transition: all 200ms
```

**Principles**
- Consistent 48px height for interactive elements
- Subtle background tint (not pure white)
- Warm glow on focus (amber ring)
- No harsh outlines

### Buttons

**Primary**
```css
width: 100%
height: 48px
background: linear-gradient(to right, stone-800, stone-950)
color: stone-50
font-size: 11px
letter-spacing: 0.15em
text-transform: uppercase
font-weight: 300
box-shadow: 0 2px 8px rgba(120,53,15,0.12)

/* Hover */
background: linear-gradient(to right, amber-900, amber-950)
transition: all 300ms
```

**Principles**
- Full width for emphasis
- Restrained default, warm hover
- Color reveals intent through interaction
- Subtle warm shadow (not black)

### Links

```css
font-size: 11px
letter-spacing: 0.15em
text-transform: uppercase
font-weight: 300
color: stone-600

/* Hover */
color: amber-900
transition: colors 200ms
```

---

## Borders & Lines

### Border Widths
```css
border: 1px  /* Standard - precise, not heavy */
```

### Border Colors
```css
stone-800/90  /* Card outer border - strong but not solid */
stone-800/80  /* Divider lines - architectural */
stone-200     /* Input borders - soft */
```

**Principles**
- Borders are sculptural elements
- Use transparency (80%, 90%) for material quality
- Horizontal lines divide space like Judd boxes
- Never use rounded corners (pure geometry)

---

## Shadows

### Light Mode (Desert Sun)
```css
/* Card Shadow - Warm, atmospheric */
box-shadow: 0 8px 32px rgba(120,53,15,0.08)

/* Button Shadow - Subtle depth */
box-shadow: 0 2px 8px rgba(120,53,15,0.12)
```

### Dark Mode (Night Depth)
```css
/* Card Shadow - Deep night */
dark:shadow-[0_8px_32px_rgba(0,0,0,0.4)]

/* Button Shadow - Black void */
dark:shadow-[0_2px_8px_rgba(0,0,0,0.3)]
```

**Principles**
- **Light**: Shadows are warm-tinted (brown, not black)
- **Dark**: Shadows are deep black (void of night sky)  
- Very subtle (8-12% opacity in light, 30-40% in dark)
- Atmospheric quality over sharp depth
- Light suggests desert sun, dark suggests starlit void

---

## Animation & Interaction

### Timing
```css
transition-duration: 300ms  /* Default */
transition-duration: 200ms  /* Quick (colors, borders) */
```

### Easing
```css
transition-timing-function: ease  /* Default cubic-bezier */
```

### Cursor (Terminal Aesthetic)
```css
width: 8px     /* 2 in Tailwind units */
height: 14px   /* 3.5 in Tailwind units */
background: stone-900
animation: pulse  /* Tailwind's built-in pulse */
```

**Principles**
- Smooth, considered transitions (300ms)
- Quick feedback for direct actions (200ms)
- Color change is primary interaction signal
- Terminal cursor adds technical character

---

## Layout Principles

### Centering
```css
min-height: 100vh
display: flex
align-items: center
justify-content: center
padding: 32px  /* Breathing room from viewport edges */
```

### Container Constraints
```css
max-width: 480px  /* Precise golden rectangle proportions */
width: 100%       /* Fluid within constraint */
```

**Principles**
- Center composition in viewport
- Precise geometric constraints (480px is deliberate)
- Generous viewport padding for mobile
- Form follows function (no arbitrary widths)

---

## Accessibility

### Focus States
```css
focus:border-amber-900/40
focus:ring-1 focus:ring-amber-900/20
focus:outline-none
```

### Color Contrast
- Stone-900 on white: >12:1 (AAA)
- Stone-600 on white: >7:1 (AAA)
- Button text (stone-50 on stone-950): >15:1 (AAA)

### Motion
- All animations respect `prefers-reduced-motion`
- Transitions are functional, not decorative

---

## Implementation Notes

### Tailwind Configuration

Add to `tailwind.config.js`:

```js
module.exports = {
  theme: {
    extend: {
      letterSpacing: {
        widest: '0.2em',
        wider: '0.15em',
      },
      fontWeight: {
        light: 300,
      },
    },
  },
}
```

### Font Loading
- System fonts only (no web fonts)
- Ensures instant rendering
- Honors minimalist principle (use what's available)

---

## Visual References

### Influences
- **Donald Judd**: Geometric precision, industrial materials, color as structure
- **Marfa, Texas**: Desert light, warm earth tones, vast negative space
- **Brutalist Architecture**: Honest materials, bold geometry, no decoration
- **Terminal Interfaces**: Monospace aesthetics, cursor elements, precision

### Mood
- Warm but restrained
- Precise but approachable
- Technical but human
- Desert minimalism

---

## Usage Guidelines

### Do
✓ Embrace white space (it's a feature, not emptiness)
✓ Use precise measurements (48px, not "around 50px")
✓ Let warmth emerge through interaction
✓ Keep geometric forms pure (no rounded corners)
✓ Use extended tracking for sculptural typography
✓ Add subtle warm tints to shadows and backgrounds

### Don't
✗ Add decoration or embellishment
✗ Use more than light (300) font weight
✗ Introduce new colors outside the palette
✗ Round corners or soften edges
✗ Reduce padding to "fit more in"
✗ Use pure black (#000) in light mode - use stone-950
✗ Use pure white (#fff) in dark mode - use stone-300
✗ Add icons unless absolutely necessary

---

## Extending the System

When adding new components:

1. **Start with geometry**: Define the pure rectangular form
2. **Add structure**: Use stone borders to divide space
3. **Apply restraint**: Minimal color in default state
4. **Design interaction**: Reveal warmth on hover/focus
5. **Consider spacing**: Generous padding is intentional
6. **Support both modes**: Light (day) and dark (night) aesthetics
7. **Test in context**: Does it feel like West Texas minimalism?

---

*This is not just a design system—it's an aesthetic philosophy applied to interface design.*
