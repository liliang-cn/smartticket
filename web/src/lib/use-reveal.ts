import { useEffect, useRef } from "react";
import gsap from "gsap";

/**
 * Staggered entrance for elements marked with [data-reveal] inside the
 * returned ref. Re-runs when `key` changes (e.g. on route/data change).
 */
export function useReveal<T extends HTMLElement = HTMLDivElement>(key?: unknown) {
  const ref = useRef<T>(null);
  useEffect(() => {
    const root = ref.current;
    if (!root) return;
    const targets = root.querySelectorAll("[data-reveal]");
    if (!targets.length) return;
    const ctx = gsap.context(() => {
      gsap.fromTo(
        targets,
        { y: 14, opacity: 0 },
        {
          y: 0,
          opacity: 1,
          duration: 0.5,
          ease: "power3.out",
          stagger: 0.05,
        }
      );
    }, root);
    return () => ctx.revert();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key]);
  return ref;
}
