import React from "react";
import { AbsoluteFill, Sequence, useCurrentFrame, interpolate, Easing, staticFile } from "remotion";
import { Audio } from "@remotion/media";
import { Background } from "../components/Background";
import { GlassPanel } from "../components/GlassPanel";
import { LogoMark } from "../components/Logo";
import { interFontFamily } from "../fonts";

export const Outro: React.FC = () => {
  const frame = useCurrentFrame();

  const logoP = interpolate(frame, [0, 24], [0, 1], {
    easing: Easing.bezier(0.16, 1, 0.3, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  const wordmarkOpacity = interpolate(frame, [14, 36], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const wordmarkY = interpolate(frame, [14, 36], [16, 0], {
    easing: Easing.bezier(0.16, 1, 0.3, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  const taglineOpacity = interpolate(frame, [30, 52], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const taglineY = interpolate(frame, [30, 52], [14, 0], {
    easing: Easing.bezier(0.16, 1, 0.3, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  const ctaOpacity = interpolate(frame, [48, 70], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const ctaScale = interpolate(frame, [48, 70], [0.92, 1], {
    easing: Easing.bezier(0.34, 1.3, 0.64, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  return (
    <AbsoluteFill style={{ fontFamily: interFontFamily }}>
      <Sequence from={45} layout="none">
        <Audio src={staticFile("sfx/ding.wav")} volume={0.3} />
      </Sequence>
      <Background
        blobs={[
          { color: "#bfe0ff", size: 900, top: "-16%", left: "60%", driftX: -25, driftY: 20 },
          { color: "#ffd7ec", size: 800, top: "48%", left: "-14%", driftX: 25, driftY: -20 },
        ]}
      />
      <AbsoluteFill style={{ alignItems: "center", justifyContent: "center" }}>
        <div style={{ display: "flex", flexDirection: "column", alignItems: "center" }}>
          <div style={{ opacity: logoP, scale: interpolate(logoP, [0, 1], [0.85, 1]) }}>
            <LogoMark size={110} />
          </div>
          <div
            style={{
              marginTop: 26,
              fontSize: 84,
              fontWeight: 700,
              color: "#0a0a0a",
              letterSpacing: "-0.01em",
              opacity: wordmarkOpacity,
              translate: `0 ${wordmarkY}px`,
            }}
          >
            Timeless
          </div>
          <div
            style={{
              marginTop: 14,
              fontSize: 30,
              fontWeight: 500,
              color: "#737373",
              opacity: taglineOpacity,
              translate: `0 ${taglineY}px`,
            }}
          >
            Close more sponsorships. Do less of the work.
          </div>

          <GlassPanel
            radius={999}
            style={{
              marginTop: 48,
              padding: "20px 46px",
              opacity: ctaOpacity,
              scale: ctaScale,
            }}
          >
            <div style={{ fontSize: 26, fontWeight: 600, color: "#0a0a0a" }}>
              Get started free
            </div>
          </GlassPanel>
        </div>
      </AbsoluteFill>
    </AbsoluteFill>
  );
};
