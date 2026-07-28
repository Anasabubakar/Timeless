import React from "react";
import { AbsoluteFill, useCurrentFrame, interpolate, Easing } from "remotion";
import { Background } from "../components/Background";
import { GlassPanel } from "../components/GlassPanel";
import { interFontFamily } from "../fonts";

const BARS = [38, 52, 46, 68, 74, 92];

const STATS = [
  { label: "Pipeline value", value: 2.4, prefix: "$", suffix: "M" },
  { label: "Win rate", value: 38, suffix: "%" },
  { label: "Avg. close time", value: 21, suffix: " days" },
];

const Counter: React.FC<{
  frame: number;
  start: number;
  target: number;
  prefix?: string;
  suffix?: string;
  decimals?: number;
}> = ({ frame, start, target, prefix = "", suffix = "", decimals }) => {
  const p = interpolate(frame, [start, start + 40], [0, 1], {
    easing: Easing.bezier(0.16, 1, 0.3, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const value = target * p;
  const formatted = decimals !== undefined ? value.toFixed(decimals) : Math.round(value).toString();
  return (
    <span>
      {prefix}
      {formatted}
      {suffix}
    </span>
  );
};

export const AnalyticsFeature: React.FC = () => {
  const frame = useCurrentFrame();

  const headingOpacity = interpolate(frame, [0, 20], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const headingY = interpolate(frame, [0, 20], [16, 0], {
    easing: Easing.bezier(0.16, 1, 0.3, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  const panelOpacity = interpolate(frame, [10, 32], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const panelY = interpolate(frame, [10, 32], [26, 0], {
    easing: Easing.bezier(0.16, 1, 0.3, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  return (
    <AbsoluteFill style={{ fontFamily: interFontFamily }}>
      <Background
        blobs={[
          { color: "#d3f0e0", size: 820, top: "-14%", left: "68%", driftX: -20, driftY: 15 },
          { color: "#d6e4ff", size: 780, top: "56%", left: "-10%", driftX: 20, driftY: -15 },
        ]}
      />
      <AbsoluteFill
        style={{ alignItems: "center", justifyContent: "center", flexDirection: "column" }}
      >
        <div
          style={{
            textAlign: "center",
            opacity: headingOpacity,
            translate: `0 ${headingY}px`,
            marginBottom: 36,
          }}
        >
          <div style={{ fontSize: 58, fontWeight: 700, color: "#0a0a0a", letterSpacing: "-0.01em" }}>
            See every dollar in motion
          </div>
          <div style={{ fontSize: 26, fontWeight: 500, color: "#737373", marginTop: 10 }}>
            Live analytics on pipeline health and revenue
          </div>
        </div>

        <GlassPanel
          style={{
            width: 1360,
            padding: "40px 48px",
            opacity: panelOpacity,
            translate: `0 ${panelY}px`,
          }}
        >
          <div style={{ display: "flex", gap: 60, marginBottom: 40 }}>
            {STATS.map((stat, i) => (
              <div key={stat.label} style={{ flex: 1 }}>
                <div style={{ fontSize: 44, fontWeight: 700, color: "#0a0a0a" }}>
                  <Counter
                    frame={frame}
                    start={20 + i * 10}
                    target={stat.value}
                    prefix={stat.prefix}
                    suffix={stat.suffix}
                    decimals={stat.value % 1 !== 0 ? 1 : undefined}
                  />
                </div>
                <div style={{ fontSize: 19, color: "#737373", fontWeight: 500, marginTop: 6 }}>
                  {stat.label}
                </div>
              </div>
            ))}
          </div>

          <div
            style={{
              display: "flex",
              alignItems: "flex-end",
              gap: 22,
              height: 200,
              borderTop: "1px solid rgba(10,10,10,0.08)",
              paddingTop: 24,
            }}
          >
            {BARS.map((h, i) => {
              const p = interpolate(frame, [50 + i * 8, 78 + i * 8], [0, 1], {
                easing: Easing.bezier(0.16, 1, 0.3, 1),
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
              });
              return (
                <div
                  key={i}
                  style={{
                    flex: 1,
                    height: `${h * p}%`,
                    borderRadius: "10px 10px 0 0",
                    background:
                      i === BARS.length - 1
                        ? "linear-gradient(180deg, #22c55e, #16a34a)"
                        : "linear-gradient(180deg, #93c5fd, #60a5fa)",
                  }}
                />
              );
            })}
          </div>
        </GlassPanel>
      </AbsoluteFill>
    </AbsoluteFill>
  );
};
