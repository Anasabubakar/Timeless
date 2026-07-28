import React from "react";

export const GlassPanel: React.FC<{
  children: React.ReactNode;
  style?: React.CSSProperties;
  radius?: number;
}> = ({ children, style, radius = 32 }) => {
  return (
    <div
      style={{
        background: "rgba(255,255,255,0.55)",
        backdropFilter: "blur(24px) saturate(180%)",
        WebkitBackdropFilter: "blur(24px) saturate(180%)",
        border: "1px solid rgba(255,255,255,0.7)",
        borderRadius: radius,
        boxShadow:
          "0 30px 80px rgba(15,15,20,0.10), 0 2px 8px rgba(15,15,20,0.04), inset 0 1px 0 rgba(255,255,255,0.8)",
        ...style,
      }}
    >
      {children}
    </div>
  );
};
