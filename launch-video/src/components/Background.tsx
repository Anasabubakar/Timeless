import React from "react";
import { AbsoluteFill, useCurrentFrame, useVideoConfig, interpolate, Easing } from "remotion";

type Blob = {
  color: string;
  size: number;
  top: string;
  left: string;
  driftX?: number;
  driftY?: number;
  delay?: number;
};

const DEFAULT_BLOBS: Blob[] = [
  { color: "#bfe0ff", size: 900, top: "-12%", left: "-10%", driftX: 40, driftY: 20 },
  { color: "#e6d3ff", size: 820, top: "38%", left: "68%", driftX: -30, driftY: 30, delay: 10 },
  { color: "#ffd7ec", size: 760, top: "62%", left: "8%", driftX: 25, driftY: -25, delay: 20 },
];

export const Background: React.FC<{ blobs?: Blob[] }> = ({ blobs = DEFAULT_BLOBS }) => {
  const frame = useCurrentFrame();
  const { durationInFrames } = useVideoConfig();

  return (
    <AbsoluteFill style={{ backgroundColor: "#fcfcfc" }}>
      {blobs.map((blob, i) => {
        const t = interpolate(
          frame - (blob.delay ?? 0),
          [0, durationInFrames],
          [0, 1],
          {
            easing: Easing.inOut(Easing.sin),
            extrapolateLeft: "clamp",
            extrapolateRight: "clamp",
          },
        );
        const x = interpolate(t, [0, 1], [0, blob.driftX ?? 0]);
        const y = interpolate(t, [0, 1], [0, blob.driftY ?? 0]);
        return (
          <div
            key={i}
            style={{
              position: "absolute",
              top: blob.top,
              left: blob.left,
              width: blob.size,
              height: blob.size,
              borderRadius: "50%",
              background: blob.color,
              filter: "blur(120px)",
              opacity: 0.55,
              translate: `${x}px ${y}px`,
            }}
          />
        );
      })}
      <AbsoluteFill
        style={{
          background:
            "radial-gradient(circle at 50% 40%, rgba(255,255,255,0) 0%, rgba(255,255,255,0.6) 75%)",
        }}
      />
    </AbsoluteFill>
  );
};
