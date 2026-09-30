// The "supermarket" style: components dressed like a classic online
// supermarket template (Inter, 14px medium body text in slate, navy
// headings, white product cards with a hairline border that turn green
// with a soft shadow on hover, compact buttons, pill badges, round
// category pictures, a light grey footer). Values follow that template
// exactly; its transitions and animations are left out on purpose.
// Colours come from the theme settings, so a merchant can still recolour.

export const supermarketCSS = `
@layer cos-components, cos-style;
@layer cos-style {
  body { font-size: 0.875rem; font-weight: 500; line-height: 1.5; color: var(--color-muted); }
  a { color: var(--color-primary); }
  a:hover { color: #088a08; }

  .cos-heading, .cos-product-name, .cos-logo { color: var(--color-text); }
  .cos-heading { font-weight: 600; line-height: 1.2; letter-spacing: -0.03rem; margin-bottom: 0.5rem; }
  h1.cos-heading { font-size: calc(1.34375rem + 1.125vw); }
  h2.cos-heading { font-size: calc(1.3rem + 0.6vw); }
  h3.cos-heading { font-size: calc(1.27813rem + 0.3375vw); }
  h4.cos-heading { font-size: calc(1.25625rem + 0.075vw); }
  h5.cos-heading { font-size: 1.09375rem; }
  h6.cos-heading { font-size: 0.875rem; }
  @media (min-width: 1200px) {
    h1.cos-heading { font-size: 2.1875rem; }
    h2.cos-heading { font-size: 1.75rem; }
    h3.cos-heading { font-size: 1.53125rem; }
    h4.cos-heading { font-size: 1.3125rem; }
  }
  .cos-text { color: var(--color-muted); }

  /* buttons: compact, 1px border, 0.375rem corners; green is for buying,
     navy (secondary) for calls to action */
  .cos-button { padding: 0.55rem 1rem; font-size: 0.875rem; font-weight: 600; line-height: 1.5; border-width: 1px; border-radius: 0.375rem; }
  .cos-button-primary:hover { background: #099309; border-color: #088a08; color: #fff; }
  .cos-button-secondary:hover { background: #000; border-color: #000; color: #fff; }
  .cos-button-outline { border-color: var(--color-primary); color: var(--color-primary); }
  .cos-button-outline:hover { background: var(--color-primary); color: #fff; }
  .cos-hero .cos-button-primary { background: var(--color-secondary); border-color: var(--color-secondary); }
  .cos-hero .cos-button-primary:hover { background: #000; border-color: #000; }

  /* inputs */
  .cos-search input, .cos-checkout input, .cos-checkout select, .cos-login input, .cos-newsletter input[type="email"], .cos-order-status input,
  .cos-variants select, .cos-coupon input {
    padding: 0.55rem 1rem; font-size: 0.875rem; border: 1px solid #e9edec; border-radius: 0.375rem; background: #fff; color: #21313c; }
  .cos-search input:focus, .cos-checkout input:focus, .cos-login input:focus, .cos-newsletter input:focus {
    outline: 0; border-color: #85d685; box-shadow: 0 0 0 0.25rem rgba(10, 173, 10, 0.25); }

  /* header */
  .cos-header { background: #fff; border-bottom: 1px solid #e9edec; }
  .cos-header .cos-section { padding-top: 1rem; padding-bottom: 1rem; }
  .cos-header .cos-banner { background: #f0f3f2; color: #5c6c75; font-size: 0.8125rem; padding: 0.25rem 1rem; }
  .cos-header .cos-banner a { color: var(--color-primary); }
  .cos-search { position: relative; }
  .cos-search button { padding: 0.55rem 1rem; font-size: 0.875rem; font-weight: 600; border-radius: 0.375rem;
                       background: var(--color-primary); color: #fff; border: 1px solid var(--color-primary); }
  .cos-nav ul { gap: 1.5rem; }
  .cos-header .cos-nav a { color: #001e2b; font-weight: 500; padding: 0.5rem 0; display: inline-block; }
  .cos-header .cos-nav a:hover, .cos-nav a:hover { color: var(--color-primary); }
  .cos-account-link, .cos-cart-link, .cos-language-switcher a { color: #5c6c75; font-weight: 500; }
  .cos-account-link:hover, .cos-cart-link:hover { color: var(--color-primary); }
  .cos-count { background: var(--color-primary); border-radius: 50rem; font-size: 0.75em; font-weight: 700; padding: 0.35em 0.65em; line-height: 1; min-width: 0; }

  /* hero: rounded picture, yellow pill, big navy title */
  .cos-hero { border-radius: 0.5rem; padding: 3.5rem 2rem; background-color: #f0f3f2; }
  @media (min-width: 992px) { .cos-hero { padding: 4rem 3rem; } }
  .cos-hero-content { max-width: 36rem; gap: 0; }
  .cos-hero-badge { display: inline-block; background: #ffc107; color: #000; font-size: 0.75em; font-weight: 700; line-height: 1;
                    padding: 0.35em 0.65em; border-radius: 0.375rem; }
  .cos-hero .cos-heading { font-size: calc(1.425rem + 2.1vw); font-weight: 700; line-height: 1.2; color: #001e2b; margin: 1rem 0 0.5rem; }
  @media (min-width: 1200px) { .cos-hero .cos-heading { font-size: 3rem; } }
  .cos-hero p { font-size: 1.09375rem; font-weight: 400; color: #5c6c75; margin-bottom: 1rem; }
  .cos-hero .cos-button { margin-top: 1rem; }

  /* product cards */
  .cos-product-grid { gap: 1.5rem; }
  .cos-product-tile { position: relative; background: #fff; border: 1px solid #e9edec; border-radius: 0.375rem; padding: 1rem; gap: 0.25rem; }
  .cos-product-tile:hover { border-color: var(--color-primary); box-shadow: 0 5px 12px rgba(0, 0, 0, 0.12); }
  .cos-product-link { gap: 0.25rem; }
  .cos-product-image { background: #fff; border-radius: 0; margin-bottom: 0.75rem; }
  .cos-product-image img { object-fit: contain; }
  .cos-product-name { color: #001e2b; font-size: 0.9375rem; font-weight: 600; line-height: 1.2; }
  .cos-product-link:hover .cos-product-name { color: var(--color-primary); }
  .cos-product-price { display: flex; flex-wrap: wrap; align-items: center; gap: 0.25rem; min-height: 1.875rem; margin-top: 0.75rem;
                       color: #001e2b; font-weight: 500; padding-right: 5rem; }
  .cos-product-price s { order: 2; color: #5c6c75; font-weight: 500; }
  .cos-product-price .cos-sale { order: 1; color: #001e2b; font-weight: 500; }
  .cos-badge-discount { top: 0; left: 0; border-radius: 0.375rem; background: #db3030; color: #fff; font-size: 0.75em; font-weight: 700;
                        padding: 0.35em 0.65em; line-height: 1; }
  .cos-product-tile .cos-tile-action { position: absolute; right: 1rem; bottom: 1rem; width: auto; margin: 0;
                                        padding: 0.25rem 0.5rem; font-size: 0.765625rem; border-radius: 0.25rem; }
  .cos-product-tile .cos-button-primary.cos-tile-action { background: var(--color-primary); border-color: var(--color-primary); }
  .cos-stock { font-size: 0.75rem; }
  .cos-price { color: #001e2b; font-size: 1.3125rem; font-weight: 600; }

  /* category tiles: round pictures, name underneath */
  .cos-collection-list { grid-template-columns: repeat(auto-fill, minmax(8.5rem, 1fr)); gap: 2.5rem 1.5rem; }
  .cos-collection-tile { align-items: center; text-align: center; gap: 1rem; color: #001e2b; font-size: 0.875rem; font-weight: 600; }
  .cos-collection-tile:hover { color: var(--color-primary); }
  .cos-collection-tile-image { width: 100%; max-width: 9rem; aspect-ratio: 1; border-radius: 50%; background: #f0f3f2; }
  .cos-collection-tile:hover .cos-collection-tile-image { box-shadow: 0 0.5rem 1rem rgba(0, 0, 0, 0.15); }

  /* listing pages */
  .cos-filters a { border-color: #e9edec; color: #5c6c75; border-radius: 0.375rem; }
  .cos-filters a:hover { color: var(--color-primary); }
  .cos-breadcrumbs ol { color: #5c6c75; }
  .cos-pagination a, .cos-pagination span { color: #5c6c75; }

  /* newsletter */
  .cos-newsletter .cos-button { background: var(--color-primary); border-color: var(--color-primary); color: #fff; }

  /* footer */
  .cos-footer { background: #f0f3f2; padding: 4rem 0; }
  .cos-footer .cos-section { padding-top: 0; padding-bottom: 0; }
  .cos-footer .cos-heading { color: #001e2b; margin-bottom: 1rem; }
  .cos-footer .cos-nav a, .cos-footer .cos-text { color: #5c6c75; }
  .cos-footer .cos-nav a:hover { color: var(--color-primary); }
  .cos-footer .cos-nav-vertical ul { gap: 0.5rem; }
  .cos-footer .cos-social a { border: 1px solid #889397; border-radius: 0.5rem; color: #889397; padding: 0.375rem; }
  .cos-footer .cos-social a:hover { border-color: var(--color-primary); color: var(--color-primary); }

  /* nothing moves */
  *, *::before, *::after { transition: none !important; animation: none !important; }
}
`
