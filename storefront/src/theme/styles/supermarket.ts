// The "supermarket" style: components dressed like a classic online
// supermarket template -- Inter, slate body text and navy headings, a grey
// top bar with line icons, centred section titles over a double orange
// rule, round grey category pictures, white hairline product cards that
// turn green with a soft shadow on hover, green "+ Add" buttons, navy
// calls to action, bordered service promises and a grey footer whose links
// carry chevrons and dividers. Values follow that template; its sliders,
// transitions and animations are left out on purpose. Colours come from
// the theme settings where the template uses its brand colours.

// Line icons drawn as CSS masks, so they take the text colour.
const svg = (body: string) =>
  `url("data:image/svg+xml,${encodeURIComponent(`<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='black' stroke-width='1.7' stroke-linecap='round' stroke-linejoin='round'>${body}</svg>`)}")`
const ICONS = {
  heart: svg("<path d='M12 20s-7.5-4.6-7.5-10.2A4.3 4.3 0 0 1 12 7a4.3 4.3 0 0 1 7.5 2.8C19.5 15.4 12 20 12 20z'/>"),
  user: svg("<circle cx='12' cy='8' r='4'/><path d='M4 21c0-4 3.6-7 8-7s8 3 8 7'/>"),
  bag: svg("<path d='M5 8h14l-1 13H6z'/><path d='M9 8V6a3 3 0 0 1 6 0v2'/>"),
  grid: svg("<rect x='4' y='4' width='6' height='6'/><rect x='14' y='4' width='6' height='6'/><rect x='4' y='14' width='6' height='6'/><rect x='14' y='14' width='6' height='6'/>"),
  chevron: svg("<path d='m9 6 6 6-6 6'/>"),
  send: svg("<path d='M21 3 3 10.5l7 3 3 7z'/><path d='m10 13.5 4-4'/>"),
}
const icon = (name: keyof typeof ICONS, size: string) =>
  `content: ""; display: inline-block; width: ${size}; height: ${size}; background-color: currentColor; -webkit-mask: ${ICONS[name]} center / contain no-repeat; mask: ${ICONS[name]} center / contain no-repeat;`

export const supermarketCSS = `
@layer cos-reset, cos-components, cos-style;
@layer cos-style {
  body { font-size: 0.875rem; font-weight: 500; line-height: 1.5; color: var(--color-muted); }
  .cos-text a { color: var(--color-primary); }

  /* type */
  .cos-heading { color: var(--color-text); font-weight: 600; line-height: 1.2; letter-spacing: -0.03rem; margin: 0 0 0.5rem; }
  h1.cos-heading { font-size: calc(1.34375rem + 1.125vw); }
  h2.cos-heading { font-size: calc(1.3rem + 0.6vw); }
  h3.cos-heading { font-size: calc(1.27813rem + 0.3375vw); }
  h4.cos-heading { font-size: 1.25rem; }
  h5.cos-heading { font-size: 1.09375rem; }
  h6.cos-heading { font-size: 0.875rem; }
  @media (min-width: 1200px) {
    h1.cos-heading { font-size: 2.1875rem; }
    h2.cos-heading { font-size: 1.75rem; }
    h3.cos-heading { font-size: 1.75rem; }
  }
  .cos-text { color: var(--color-muted); font-size: 1rem; }

  /* sections: centred titles over a double orange rule */
  .cos-section { padding-top: 2.5rem; padding-bottom: 2.5rem; }
  .cos-main > .cos-section:first-child, .cos-main .cos-section:first-child { padding-top: 2.25rem; }
  .cos-section > h3.cos-heading:first-child { text-align: center; margin-bottom: 2.5rem; }
  .cos-section > h3.cos-heading:first-child::after {
    content: ""; display: block; width: 60px; height: 8px; margin: 0.75rem auto 0;
    background: linear-gradient(#f1a228, #f1a228) center top / 60px 2px no-repeat, linear-gradient(#f1a228, #f1a228) center bottom / 40px 2px no-repeat; }

  /* buttons: green to buy, navy calls to action */
  .cos-button { padding: 0.625rem 1.25rem; font-size: 1rem; font-weight: 600; line-height: 1.5; border-width: 1px; border-radius: 0.375rem; text-decoration: none; }
  .cos-button-primary { background: var(--color-primary); border-color: var(--color-primary); color: #fff; }
  .cos-button-primary:hover { background: #067306; border-color: #067306; color: #fff; }
  .cos-button-secondary { background: var(--color-secondary); border-color: var(--color-secondary); color: #fff; }
  .cos-button-secondary:hover { background: #000; border-color: #000; color: #fff; }
  .cos-button-outline { border-color: var(--color-primary); color: var(--color-primary); }
  .cos-button-outline:hover { background: var(--color-primary); color: #fff; }
  .cos-column > .cos-button { align-self: flex-start; }

  /* inputs */
  .cos-search input, .cos-checkout input, .cos-checkout select, .cos-login input, .cos-newsletter input[type="email"], .cos-order-status input,
  .cos-variants select, .cos-coupon input {
    padding: 0.55rem 1rem; font-size: 1rem; border: 1px solid #dfe2e1; border-radius: 0.375rem; background: #fff; color: #21313c; }
  .cos-search input:focus, .cos-checkout input:focus, .cos-login input:focus, .cos-newsletter input:focus {
    outline: 0; border-color: #85d685; box-shadow: 0 0 0 0.25rem rgba(8, 138, 8, 0.25); }
  .cos-search input::placeholder, .cos-newsletter input::placeholder { color: #5c6c75; }

  /* header: grey top bar with line icons; logo, search and menu below */
  .cos-header { background: #fff; box-shadow: 0 2px 4px rgba(0, 30, 43, 0.06); }
  .cos-header .cos-section { padding-top: 0.75rem; padding-bottom: 0.75rem; }
  .cos-header .cos-section:first-child { background: #f0f3f2; padding-top: 0.5rem; padding-bottom: 0.5rem; }
  .cos-header .cos-section:first-child .cos-text { font-size: 1rem; color: #5c6c75; }
  @media (max-width: 767px) { .cos-header .cos-section:first-child .cos-text { font-size: 0.875rem; } }
  .cos-header .cos-section + .cos-section { border-top: 1px solid #e9edec; padding-top: 1rem; padding-bottom: 1rem; }
  .cos-header .cos-columns { align-items: center; gap: 1.5rem; }
  .cos-header .cos-columns > .cos-column:last-child { flex-direction: row; justify-content: flex-end; align-items: center; gap: 1.25rem; }
  .cos-header .cos-search button { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); }
  .cos-header .cos-nav ul { gap: 1.25rem; flex-wrap: wrap; justify-content: flex-end; }
  .cos-header .cos-nav a { color: #001e2b; font-size: 1rem; font-weight: 500; display: inline-flex; align-items: center; gap: 0.5rem; }
  .cos-header .cos-nav a:hover { color: var(--color-primary); }
  .cos-header .cos-section:last-child .cos-nav li:first-child a::before { ${icon('grid', '1rem')} }
  .cos-header .cos-section:first-child .cos-nav a[href$="/wishlist"], .cos-header .cos-account-link, .cos-header .cos-cart-link {
    position: relative; font-size: 0; color: #001e2b; display: inline-flex; }
  .cos-header .cos-section:first-child .cos-nav a[href$="/wishlist"]::before { ${icon('heart', '1.375rem')} }
  @media (pointer: fine) { .cos-header .cos-section:first-child .cos-nav a[href$="/wishlist"] { min-width: 0; } }
  .cos-header .cos-account-link::before { ${icon('user', '1.375rem')} }
  .cos-header .cos-cart-link::before { ${icon('bag', '1.375rem')} }
  .cos-header .cos-count { position: absolute; top: -0.55rem; right: -0.7rem; font-size: 0.6875rem; font-weight: 700; line-height: 1;
    padding: 0.3em 0.5em; border-radius: 50rem; background: #198754; color: #fff; }
  .cos-header .cos-language-switcher a { color: #001e2b; font-size: 0.875rem; }
  .cos-logo { color: var(--color-primary); font-weight: 800; letter-spacing: -0.02em; }

  /* hero: tinted panel, yellow label, big navy title, navy button */
  .cos-hero { border-radius: 0.5rem; padding: 4rem 3rem; background-color: #d3eadd; }
  @media (min-width: 992px) { .cos-hero { padding: 5rem 4.5rem; } .cos-hero-lg { min-height: 40rem; } }
  @media (max-width: 767px) { .cos-hero { padding: 2.5rem 1.5rem; } .cos-hero p { font-size: 1.125rem; } }
  .cos-hero-content { max-width: 34rem; gap: 0; }
  .cos-hero[style*="background-image"] .cos-hero-content { padding: 0; background: none; }
  /* On narrow screens the photo sits behind the text: fade it under the copy. */
  @media (max-width: 767px) {
    .cos-hero { position: relative; isolation: isolate; background-position: 75% center; }
    .cos-hero::before { content: ""; position: absolute; inset: 0; z-index: -1; border-radius: inherit;
      background: linear-gradient(180deg, rgba(211, 234, 221, 0.96) 0%, rgba(211, 234, 221, 0.9) 70%, rgba(211, 234, 221, 0.4) 100%); }
    .cos-header .cos-nav ul { justify-content: flex-start; }
    .cos-header .cos-columns > .cos-column:last-child { justify-content: flex-start; }
  }
  .cos-hero-badge { display: inline-block; align-self: flex-start; background: #ffc107; color: #000; font-size: 0.75rem; font-weight: 700; line-height: 1;
                    padding: 0.35em 0.65em; border-radius: 0.375rem; }
  .cos-hero .cos-heading { font-size: calc(1.425rem + 2.1vw); font-weight: 700; line-height: 1.2; color: #001e2b; margin: 1.5rem 0 1rem; }
  @media (min-width: 1200px) { .cos-hero .cos-heading { font-size: 3.25rem; } }
  .cos-hero p { font-size: 1.5rem; font-weight: 400; line-height: 1.6; color: #5c6c75; }
  .cos-hero .cos-button-primary { margin-top: 2rem; background: var(--color-secondary); border-color: var(--color-secondary); padding: 0.75rem 1.25rem; }
  .cos-hero .cos-button-primary:hover { background: #000; border-color: #000; }

  /* category tiles: round grey pictures, name underneath, six to a row */
  .cos-collection-list { grid-template-columns: repeat(6, minmax(0, 1fr)); gap: 3rem 1.5rem; }
  @media (max-width: 991px) { .cos-collection-list { grid-template-columns: repeat(3, minmax(0, 1fr)); } }
  .cos-collection-tile { align-items: center; text-align: center; gap: 1.5rem; color: #001e2b; font-size: 1rem; font-weight: 600; }
  .cos-collection-tile:hover { color: var(--color-primary); }
  .cos-collection-tile-image { width: 8rem; max-width: 100%; aspect-ratio: 1; border-radius: 50%; background: #f0f3f2; padding: 1.5rem; }
  .cos-collection-tile-image img { object-fit: contain; }
  .cos-collection-tile:hover .cos-collection-tile-image { box-shadow: 0 0.5rem 1rem rgba(0, 0, 0, 0.15); }

  /* product cards */
  .cos-product-grid { gap: 1.5rem; }
  .cos-product-tile { position: relative; background: #fff; border: 1px solid #e9edec; border-radius: 0.5rem; padding: 1.25rem; gap: 0.5rem; }
  .cos-product-tile:hover { border-color: var(--color-primary); box-shadow: 0 5px 12px rgba(0, 0, 0, 0.12); }
  .cos-product-link { gap: 0.375rem; color: inherit; }
  .cos-product-image { background: #fff; border-radius: 0; margin: 0 auto 1rem; width: 70%; }
  .cos-product-image img { object-fit: contain; }
  .cos-product-category { color: #5c6c75; font-size: 0.875rem; font-weight: 500; }
  .cos-product-name { color: #001e2b; font-size: 1rem; font-weight: 600; line-height: 1.3; }
  .cos-product-link:hover .cos-product-name { color: var(--color-primary); }
  .cos-product-price { display: flex; flex-wrap: wrap; align-items: center; gap: 0.25rem; margin-top: 1rem; color: #001e2b; font-size: 1rem; font-weight: 500; }
  .cos-product-price s { order: 2; color: #5c6c75; font-weight: 500; }
  .cos-product-price .cos-sale { order: 1; color: #001e2b; font-weight: 500; }
  .cos-badge-discount { top: 0; left: 0; border-radius: 0.375rem; background: #db3030; color: #fff; font-size: 0.75rem; font-weight: 700;
                        padding: 0.35em 0.65em; line-height: 1; }
  .cos-tile-action { margin-top: 1rem; padding: 0.625rem 1rem; font-size: 1rem; }
  .cos-tile-compact .cos-product-price { padding-right: 4.75rem; min-height: 2.125rem; font-size: 0.9375rem; }
  .cos-tile-compact .cos-tile-action { position: absolute; right: 1.25rem; bottom: 1.25rem; width: auto; margin: 0; padding: 0.3rem 0.75rem; font-size: 0.875rem; }
  @media (pointer: fine) { .cos-tile-compact .cos-tile-action { min-height: 2.125rem; } }
  .cos-stock { font-size: 0.75rem; }
  .cos-price { color: #001e2b; font-size: 1.3125rem; font-weight: 600; }

  /* listing pages */
  .cos-filters a { border-color: #dfe2e1; color: #5c6c75; border-radius: 0.375rem; }
  .cos-filters a:hover { color: var(--color-primary-text); }
  .cos-filters a[aria-current="true"] { color: var(--color-on-primary); }
  .cos-price .cos-sale { color: #001e2b; }
  .cos-breadcrumbs ol { color: #5c6c75; }

  /* footer: grey, big titles, chevron links over hairline dividers */
  .cos-footer { background: #f0f3f2; padding: 4rem 0 3rem; }
  .cos-footer .cos-section { padding-top: 0; padding-bottom: 0; }
  .cos-footer .cos-heading { color: #001e2b; font-size: 1.75rem; font-weight: 600; margin-bottom: 0.5rem; }
  .cos-footer .cos-text { color: #001e2b; font-size: 1.0625rem; line-height: 1.7; }
  .cos-footer .cos-logo { font-size: 1.75rem; }
  .cos-footer .cos-nav-vertical ul { gap: 0; }
  .cos-footer .cos-nav-vertical li { border-bottom: 1px solid #e3e7e6; padding: 0.4rem 0; }
  .cos-footer .cos-nav a { display: inline-flex; align-items: center; gap: 0.5rem; color: #001e2b; font-size: 1.0625rem; }
  .cos-footer .cos-nav a::before { ${icon('chevron', '0.75rem')} }
  .cos-footer .cos-nav a:hover { color: var(--color-primary); }
  .cos-footer .cos-newsletter { margin-top: 1.5rem; gap: 0; }
  .cos-footer .cos-newsletter .cos-label, .cos-footer .cos-newsletter .cos-consent { display: none; }
  .cos-footer .cos-newsletter-row { gap: 2.5rem; align-items: stretch; }
  .cos-footer .cos-newsletter input[type="email"] { border-radius: 0; border-color: #5c6c75; background: #fff; }
  .cos-footer .cos-newsletter .cos-button { font-size: 0; width: 4.25rem; border-radius: 0; background: #000; border-color: #000; color: #fff; opacity: 1; }
  .cos-footer .cos-newsletter .cos-button::before { ${icon('send', '1.75rem')} }
  .cos-footer .cos-social { gap: 1.25rem; margin-top: 1rem; }
  .cos-footer .cos-social a { width: 3.4rem; height: 3.4rem; justify-content: center; border: 1px solid #c1c7c6; border-radius: 0; color: #001e2b; }
  .cos-footer .cos-social a:hover { border-color: var(--color-primary); color: var(--color-primary); }

  /* nothing moves */
  *, *::before, *::after { transition: none !important; animation: none !important; }
}
`
