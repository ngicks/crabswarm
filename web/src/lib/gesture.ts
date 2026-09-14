// Pointer-event pan/zoom gestures for a pan-and-zoom surface: one pointer
// pans, two pointers pinch. Consumers own the transform and the clamping; this
// only turns raw pointer events into (dx, dy) pan and (cx, cy, factor) zoom
// deltas in client coordinates.

export interface GestureHandlers {
  /** Move the content by (dx, dy) viewport pixels. */
  onPan(dx: number, dy: number): void;
  /** Scale the content by `factor` keeping the viewport point (cx, cy) fixed. */
  onZoom(cx: number, cy: number, factor: number): void;
  /** Pointer went down and up with no movement past the slop and no second pointer during the gesture. */
  onTap?(e: PointerEvent): void;
  /** First movement past the slop with one pointer. */
  onDragStart?(e: PointerEvent): void;
  /** Every pointer lifted or cancelled. */
  onGestureEnd?(): void;
}

export interface GestureOptions {
  /** Pixels a pointer may travel before the gesture counts as a drag. Default 3. */
  slop?: number;
  /**
   * "down": setPointerCapture on pointerdown and preventDefault (the lightbox).
   * "drag": capture on the first movement past the slop (a box where a click
   * must still reach the element under it). Default "down".
   */
  captureOn?: "down" | "drag";
}

interface TrackedPointer {
  /** Last seen position, the origin of the next pan delta. */
  x: number;
  y: number;
  /** Where the pointer went down, measured against for the slop. */
  sx: number;
  sy: number;
  captured: boolean;
}

/** Attaches pointer listeners to `el`; returns the detach function. */
export function attachGestures(el: HTMLElement, h: GestureHandlers, opts?: GestureOptions): () => void {
  const slop = opts?.slop ?? 3;
  const captureOn = opts?.captureOn ?? "down";

  const pointers = new Map<number, TrackedPointer>();
  // Gesture-wide, reset when a new gesture starts: a pinch is never a tap even
  // when both fingers land and lift without moving, and onDragStart owes the
  // consumer exactly one call per gesture.
  let moved = false;
  let hadSecond = false;
  let dragStarted = false;

  const onPointerDown = (e: PointerEvent): void => {
    if (e.button !== 0) return;
    // A third finger neither joins the pair nor disturbs it, so it must not
    // take capture or swallow the default action either.
    if (pointers.size >= 2) return;
    if (pointers.size === 0) {
      moved = false;
      hadSecond = false;
      dragStarted = false;
    } else {
      hadSecond = true;
    }
    let captured = false;
    if (captureOn === "down") {
      el.setPointerCapture(e.pointerId);
      captured = true;
      e.preventDefault();
    }
    pointers.set(e.pointerId, { x: e.clientX, y: e.clientY, sx: e.clientX, sy: e.clientY, captured });
  };

  const onPointerMove = (e: PointerEvent): void => {
    const p = pointers.get(e.pointerId);
    if (!p) return;
    const past = Math.hypot(e.clientX - p.sx, e.clientY - p.sy) > slop;
    if (past) {
      moved = true;
      if (!p.captured && captureOn === "drag") {
        p.captured = true;
        el.setPointerCapture(e.pointerId);
      }
    }

    if (pointers.size === 1) {
      if (past && !dragStarted) {
        dragStarted = true;
        h.onDragStart?.(e);
      }
      const dx = e.clientX - p.x;
      const dy = e.clientY - p.y;
      p.x = e.clientX;
      p.y = e.clientY;
      h.onPan(dx, dy);
      return;
    }

    const [a, b] = [...pointers.values()];
    // Snapshot the old midpoint and spread as numbers: `a` and `b` alias the
    // map entries, so moving the pointer below rewrites them in place.
    const mx = (a.x + b.x) / 2;
    const my = (a.y + b.y) / 2;
    const d = Math.hypot(a.x - b.x, a.y - b.y);
    p.x = e.clientX;
    p.y = e.clientY;
    const mx2 = (a.x + b.x) / 2;
    const my2 = (a.y + b.y) / 2;
    const d2 = Math.hypot(a.x - b.x, a.y - b.y);

    // Zoom around the PREVIOUS midpoint, then pan by the midpoint's own
    // movement. That keeps whatever sat between the fingers pinned there while
    // the pan carries it to where the fingers now are; scaling around the new
    // midpoint instead lets the pinned point drift a little on every event.
    // The ratio goes out raw — clamping the scale is the consumer's job.
    if (d > 0) h.onZoom(mx, my, d2 / d);
    h.onPan(mx2 - mx, my2 - my);
  };

  const release = (e: PointerEvent): void => {
    // A touch pointer is implicitly captured by the descendant it landed on.
    // When the drag-time setPointerCapture moves that capture up to `el`, the
    // browser fires lostpointercapture on the descendant, and it bubbles here
    // with a finger that is still down. A genuine loss — the pointer lifted or
    // cancelled while `el` held it — fires on `el` itself.
    if (e.type === "lostpointercapture" && e.target !== el) return;
    // An untracked id is a full no-op — no tap, no gesture end. Two ways in:
    // lostpointercapture trails the pointerup that already dropped the id, and
    // a child that stopped propagation on pointerdown (a toolbar button) still
    // lets its pointerup bubble up here.
    if (!pointers.delete(e.pointerId)) return;
    if (pointers.size > 0) {
      // Dropping back to one pointer: the survivor's stored position is
      // already its own last position, so the next pan delta starts from
      // where that finger actually is instead of jumping.
      return;
    }
    if (e.type === "pointerup" && !hadSecond && !moved) h.onTap?.(e);
    h.onGestureEnd?.();
  };

  el.addEventListener("pointerdown", onPointerDown);
  el.addEventListener("pointermove", onPointerMove);
  el.addEventListener("pointerup", release);
  el.addEventListener("pointercancel", release);
  el.addEventListener("lostpointercapture", release);
  return () => {
    el.removeEventListener("pointerdown", onPointerDown);
    el.removeEventListener("pointermove", onPointerMove);
    el.removeEventListener("pointerup", release);
    el.removeEventListener("pointercancel", release);
    el.removeEventListener("lostpointercapture", release);
  };
}
