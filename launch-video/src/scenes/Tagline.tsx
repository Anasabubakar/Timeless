import React from "react";
import { AbsoluteFill, useCurrentFrame, interpolate, Easing } from "remotion";
import { Background } from "../components/Background";
import { TransitionWhoosh } from "../components/TransitionWhoosh";
import { interFontFamily } from "../fonts";

const LINES = ["Every sponsor.", "Every stage.", "One intelligent pipeline."];

const Word: React.FC<{ text: string; start: number; frame: number }> = ({
  text,
  start,
  frame,
}) => {
  const p = interpolate(frame, [start, start + 18], [0, 1], {
    easing: Easing.bezier(0.16, 1, 0.3, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const y = interpolate(p, [0, 1], [26, 0]);
  const blur = interpolate(p, [0, 1], [8, 0]);
  return (
    <span
      style={{
        display: "inline-block",
        opacity: p,
        translate: `0 ${y}px`,
        filter: `blur(${blur}px)`,
        marginRight: "0.28em",
      }}
    >
      {text}
    </span>
  );
};

export const Tagline: React.FC = () => {
  const frame = useCurrentFrame();

  return (
    <AbsoluteFill style={{ fontFamily: interFontFamily }}>
      <TransitionWhoosh />
      <Background
        blobs={[
          { color: "#d3e0ff", size: 820, top: "6%", left: "72%", driftX: -20, driftY: 20 },
          { color: "#ffe0d3", size: 760, top: "60%", left: "-6%", driftX: 20, driftY: -15 },
        ]}
      />
      <AbsoluteFill
        style={{
          alignItems: "center",
          justifyContent: "center",
          padding: "0 160px",
        }}
      >
        <div style={{ textAlign: "center" }}>
          {LINES.map((line, i) => {
            const words = line.split(" ");
            const lineStart = 6 + i * 22;
            return (
              <div
                key={i}
                style={{
                  fontSize: 88,
                  fontWeight: 700,
                  color: i === LINES.length - 1 ? "#0a0a0a" : "#a3a3a3",
                  lineHeight: 1.16,
                  letterSpacing: "-0.01em",
                }}
              >
                {words.map((w, j) => (
                  <Word
                    key={j}
                    text={w}
                    start={lineStart + j * 5}
                    frame={frame}
                  />
                ))}
              </div>
            );
          })}
        </div>
      </AbsoluteFill>
    </AbsoluteFill>
  );
};
