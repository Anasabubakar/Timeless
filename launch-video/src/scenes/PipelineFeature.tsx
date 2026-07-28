import React from "react";
import { AbsoluteFill, useCurrentFrame, interpolate, Easing } from "remotion";
import { Background } from "../components/Background";
import { GlassPanel } from "../components/GlassPanel";
import { TransitionWhoosh } from "../components/TransitionWhoosh";
import { interFontFamily } from "../fonts";

const COLUMNS: { title: string; cards: { name: string; tag: string; color: string }[] }[] = [
  {
    title: "New",
    cards: [
      { name: "Northwind Beverages", tag: "$40K", color: "#a3a3a3" },
      { name: "Atlas Fitness", tag: "$18K", color: "#a3a3a3" },
    ],
  },
  {
    title: "Qualified",
    cards: [{ name: "Solstice Bank", tag: "$65K", color: "#60a5fa" }],
  },
  {
    title: "Proposal Sent",
    cards: [{ name: "Vantage Motors", tag: "$120K", color: "#a78bfa" }],
  },
  {
    title: "Won",
    cards: [{ name: "Aurora Foods", tag: "$85K", color: "#22c55e" }],
  },
];

const Heading: React.FC<{ frame: number }> = ({ frame }) => {
  const opacity = interpolate(frame, [0, 20], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const y = interpolate(frame, [0, 20], [16, 0], {
    easing: Easing.bezier(0.16, 1, 0.3, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  return (
    <div
      style={{
        textAlign: "center",
        opacity,
        translate: `0 ${y}px`,
        marginBottom: 44,
      }}
    >
      <div style={{ fontSize: 58, fontWeight: 700, color: "#0a0a0a", letterSpacing: "-0.01em" }}>
        A pipeline that qualifies itself
      </div>
      <div style={{ fontSize: 26, fontWeight: 500, color: "#737373", marginTop: 10 }}>
        AI scores, sorts, and moves every sponsor for you
      </div>
    </div>
  );
};

const Card: React.FC<{
  name: string;
  tag: string;
  color: string;
  frame: number;
  start: number;
  badge?: boolean;
}> = ({ name, tag, color, frame, start, badge }) => {
  const p = interpolate(frame, [start, start + 16], [0, 1], {
    easing: Easing.bezier(0.16, 1, 0.3, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const badgeP = interpolate(frame, [start + 30, start + 46], [0, 1], {
    easing: Easing.bezier(0.34, 1.4, 0.64, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  return (
    <div
      style={{
        opacity: p,
        translate: `0 ${interpolate(p, [0, 1], [14, 0])}px`,
        background: "rgba(255,255,255,0.75)",
        border: "1px solid rgba(0,0,0,0.06)",
        borderRadius: 16,
        padding: "16px 18px",
        boxShadow: "0 8px 20px rgba(15,15,20,0.05)",
        position: "relative",
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
        <div style={{ width: 10, height: 10, borderRadius: "50%", background: color }} />
        <div style={{ fontSize: 21, fontWeight: 600, color: "#0a0a0a" }}>{name}</div>
      </div>
      <div style={{ fontSize: 17, color: "#737373", marginTop: 6, marginLeft: 20 }}>{tag}</div>
      {badge && (
        <div
          style={{
            position: "absolute",
            top: -12,
            right: -10,
            background: "#0a0a0a",
            color: "#fff",
            fontSize: 13,
            fontWeight: 600,
            padding: "5px 10px",
            borderRadius: 999,
            opacity: badgeP,
            scale: interpolate(badgeP, [0, 1], [0.7, 1]),
          }}
        >
          AI qualified
        </div>
      )}
    </div>
  );
};

export const PipelineFeature: React.FC = () => {
  const frame = useCurrentFrame();
  const panelOpacity = interpolate(frame, [0, 22], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const panelY = interpolate(frame, [0, 22], [24, 0], {
    easing: Easing.bezier(0.16, 1, 0.3, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  return (
    <AbsoluteFill style={{ fontFamily: interFontFamily }}>
      <TransitionWhoosh />
      <Background
        blobs={[
          { color: "#c9e4ff", size: 850, top: "-14%", left: "-10%", driftX: 20, driftY: 15 },
          { color: "#ffe2cf", size: 760, top: "58%", left: "82%", driftX: -20, driftY: -10 },
        ]}
      />
      <AbsoluteFill style={{ alignItems: "center", justifyContent: "center" }}>
        <Heading frame={frame} />
        <GlassPanel
          style={{
            width: 1540,
            padding: "36px 40px",
            opacity: panelOpacity,
            translate: `0 ${panelY}px`,
          }}
        >
          <div style={{ display: "flex", gap: 24 }}>
            {COLUMNS.map((col, ci) => (
              <div key={col.title} style={{ flex: 1, display: "flex", flexDirection: "column", gap: 14 }}>
                <div
                  style={{
                    fontSize: 18,
                    fontWeight: 600,
                    color: "#525252",
                    textTransform: "uppercase",
                    letterSpacing: "0.06em",
                    marginBottom: 4,
                  }}
                >
                  {col.title}
                </div>
                {col.cards.map((card, cardIdx) => (
                  <Card
                    key={card.name}
                    {...card}
                    frame={frame}
                    start={24 + ci * 14 + cardIdx * 10}
                    badge={col.title === "Qualified"}
                  />
                ))}
              </div>
            ))}
          </div>
        </GlassPanel>
      </AbsoluteFill>
    </AbsoluteFill>
  );
};
