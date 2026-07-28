import React from "react";
import {
  AbsoluteFill,
  useCurrentFrame,
  useVideoConfig,
  interpolate,
  Easing,
} from "remotion";
import { Audio } from "@remotion/media";
import { staticFile } from "remotion";
import { Background } from "../components/Background";
import { LogoMark } from "../components/Logo";
import { interFontFamily } from "../fonts";

export const LogoIntro: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const reveal = interpolate(frame, [0, 26], [0, 1], {
    easing: Easing.bezier(0.16, 1, 0.3, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const scale = interpolate(reveal, [0, 1], [0.82, 1]);
  const blurPx = interpolate(reveal, [0, 1], [10, 0]);

  const haloScale = interpolate(frame, [0, 40], [0.6, 1], {
    easing: Easing.bezier(0.16, 1, 0.3, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const haloOpacity = interpolate(frame, [0, 20], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  const shineX = interpolate(frame, [18, 55], [-140, 140], {
    easing: Easing.bezier(0.45, 0, 0.55, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const shineOpacity = interpolate(
    frame,
    [18, 30, 45, 55],
    [0, 0.9, 0.9, 0],
    { extrapolateLeft: "clamp", extrapolateRight: "clamp" },
  );

  const wordmarkOpacity = interpolate(frame, [34, 58], [0, 1], {
    easing: Easing.bezier(0.16, 1, 0.3, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const wordmarkY = interpolate(frame, [34, 58], [18, 0], {
    easing: Easing.bezier(0.16, 1, 0.3, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const tracking = interpolate(frame, [34, 66], [0.35, 0.01], {
    easing: Easing.bezier(0.16, 1, 0.3, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  const taglineOpacity = interpolate(frame, [56, 78], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const taglineY = interpolate(frame, [56, 78], [10, 0], {
    easing: Easing.bezier(0.16, 1, 0.3, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  return (
    <AbsoluteFill style={{ fontFamily: interFontFamily }}>
      <Audio src={staticFile("sfx/ding.wav")} volume={0.35} />
      <Background
        blobs={[
          { color: "#bfe0ff", size: 900, top: "-16%", left: "58%", driftX: -30, driftY: 20 },
          { color: "#ffd7ec", size: 800, top: "40%", left: "-16%", driftX: 25, driftY: -20 },
        ]}
      />
      <AbsoluteFill
        style={{ alignItems: "center", justifyContent: "center" }}
      >
        <div
          style={{
            position: "relative",
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
          }}
        >
          <div
            style={{
              position: "absolute",
              width: 420,
              height: 420,
              borderRadius: "50%",
              top: -130,
              background:
                "radial-gradient(circle, rgba(255,255,255,0.9) 0%, rgba(255,255,255,0.35) 55%, rgba(255,255,255,0) 75%)",
              filter: "blur(2px)",
              scale: haloScale,
              opacity: haloOpacity,
            }}
          />
          <div
            style={{
              position: "relative",
              overflow: "hidden",
              scale,
              filter: `blur(${blurPx}px)`,
              opacity: reveal,
            }}
          >
            <LogoMark size={190} />
            <div
              style={{
                position: "absolute",
                top: 0,
                left: 0,
                width: "60%",
                height: "100%",
                background:
                  "linear-gradient(115deg, rgba(255,255,255,0) 30%, rgba(255,255,255,0.85) 50%, rgba(255,255,255,0) 70%)",
                translate: `${shineX}px 0`,
                opacity: shineOpacity,
                mixBlendMode: "overlay",
              }}
            />
          </div>
          <div
            style={{
              marginTop: 34,
              fontSize: 76,
              fontWeight: 700,
              color: "#0a0a0a",
              letterSpacing: `${tracking}em`,
              opacity: wordmarkOpacity,
              translate: `0 ${wordmarkY}px`,
            }}
          >
            Timeless
          </div>
          <div
            style={{
              marginTop: 18,
              fontSize: 30,
              fontWeight: 500,
              color: "#737373",
              letterSpacing: "0.01em",
              opacity: taglineOpacity,
              translate: `0 ${taglineY}px`,
            }}
          >
            Sponsorship intelligence, automated.
          </div>
        </div>
      </AbsoluteFill>
    </AbsoluteFill>
  );
};
