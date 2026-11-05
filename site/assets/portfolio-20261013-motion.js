// The reference's three-quarter tilt, scoped to the existing hero figure.
// Stable layout bounds avoid feedback from measuring a transformed surface.
(() => {
  const card = document.querySelector('.hero-visual');
  if (!card) return;
  const enabled = matchMedia('screen and (min-width:861px) and (hover:hover) and (pointer:fine) and (prefers-reduced-motion:no-preference)');
  let frame = 0;
  let pointer = null;
  const reset = () => {
    cancelAnimationFrame(frame);
    frame = 0;
    pointer = null;
    card.style.removeProperty('transform');
    card.style.removeProperty('transition');
  };
  const mouse = event => enabled.matches && event.pointerType === 'mouse';
  card.addEventListener('pointerenter', event => {
    if (!mouse(event)) return;
    card.style.transition = 'transform 0.2s ease-out';
  });
  card.addEventListener('pointermove', event => {
    if (!mouse(event)) { reset(); return; }
    // A media/scroll reset may occur while the pointer is still inside, with
    // no new pointerenter. Resume the reference response speed on movement.
    card.style.transition = 'transform 0.2s ease-out';
    pointer = { x: event.clientX, y: event.clientY };
    if (frame) return;
    frame = requestAnimationFrame(() => {
      frame = 0;
      if (!enabled.matches || !pointer) return;
      const parent = card.offsetParent.getBoundingClientRect();
      const left = parent.left + card.offsetLeft;
      const top = parent.top + card.offsetTop;
      const px = Math.max(-.5, Math.min(.5, (pointer.x - left) / card.offsetWidth - .5));
      const py = Math.max(-.5, Math.min(.5, (pointer.y - top) / card.offsetHeight - .5));
      card.style.transform = `perspective(900px) rotateY(${-18 + px * 14}deg) rotateX(${12 - py * 12}deg)`;
    });
  });
  card.addEventListener('pointerleave', reset);
  card.addEventListener('pointercancel', reset);
  enabled.addEventListener('change', reset);
  window.addEventListener('blur', reset);
  window.addEventListener('resize', reset);
  // Reset stale cursor coordinates when scrolling moves the card under the pointer.
  window.addEventListener('scroll', reset, { passive: true });
})();
