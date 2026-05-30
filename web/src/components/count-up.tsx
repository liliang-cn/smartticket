import { useEffect, useRef } from "react";
import gsap from "gsap";

export function CountUp({ value }: { value: number }) {
  const ref = useRef<HTMLSpanElement>(null);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const obj = { n: 0 };
    const tween = gsap.to(obj, {
      n: value,
      duration: 1,
      ease: "power2.out",
      onUpdate: () => {
        el.textContent = Math.round(obj.n).toLocaleString();
      },
    });
    return () => {
      tween.kill();
    };
  }, [value]);
  return <span ref={ref}>0</span>;
}
