import React from "react";
import { AbsoluteFill, Sequence, useCurrentFrame, interpolate, Easing, staticFile } from "remotion";
import { Audio } from "@remotion/media";
import { Background } from "../components/Background";
import { GlassPanel } from "../components/GlassPanel";
import { TransitionWhoosh } from "../components/TransitionWhoosh";
import { interFontFamily } from "../fonts";

const BODY =
  "Solstice Bank reaches 2.4M engaged fans through Aurora Foods' summer tour. This proposal outlines a $65,000 title-sponsorship package with on-site branding, digital placements, and a co-hosted fan activation.";

const TYPE_START = 46;
const TYPE_END = 150;

export const ProposalFeature: React.FC = () => {
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

  const typeProgress = interpolate(frame, [TYPE_START, TYPE_END], [0, 1], {
    easing: Easing.linear,
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });
  const charCount = Math.floor(BODY.length * typeProgress);
  const visibleText = BODY.slice(0, charCount);
  const cursorOn = frame % 20 < 10 && charCount < BODY.length;

  const barProgress = interpolate(frame, [TYPE_START, TYPE_END], [0, 100], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  const doneP = interpolate(frame, [TYPE_END, TYPE_END + 16], [0, 1], {
    easing: Easing.bezier(0.34, 1.4, 0.64, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  return (
    <AbsoluteFill style={{ fontFamily: interFontFamily }}>
      <TransitionWhoosh />
      <Sequence from={TYPE_START} layout="none">
        <Audio src={staticFile("sfx/mouse-click.wav")} volume={0.3} />
      </Sequence>
      <Background
        blobs={[
          { color: "#ffe1c9", size: 820, top: "-10%", left: "-12%", driftX: 20, driftY: 15 },
          { color: "#cfe0ff", size: 900, top: "50%", left: "72%", driftX: -20, driftY: -15 },
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
            Proposals, written while you watch
          </div>
          <div style={{ fontSize: 26, fontWeight: 500, color: "#737373", marginTop: 10 }}>
            One click turns research into a sponsor-ready deck
          </div>
        </div>

        <GlassPanel
          style={{
            width: 1180,
            padding: "40px 48px",
            opacity: panelOpacity,
            translate: `0 ${panelY}px`,
          }}
        >
          <div
            style={{
              display: "flex",
              alignItems: "center",
              justifyContent: "space-between",
              marginBottom: 22,
            }}
          >
            <div style={{ fontSize: 26, fontWeight: 700, color: "#0a0a0a" }}>
              Aurora Foods x Solstice Bank
            </div>
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: 8,
                fontSize: 16,
                fontWeight: 600,
                color: doneP > 0.5 ? "#22c55e" : "#525252",
                opacity: interpolate(frame, [TYPE_START - 4, TYPE_START], [0, 1], {
                  extrapolateLeft: "clamp",
                  extrapolateRight: "clamp",
                }),
              }}
            >
              {doneP > 0.5 ? "Ready to send" : "AI drafting"}
            </div>
          </div>

          <div
            style={{
              height: 6,
              borderRadius: 999,
              background: "rgba(10,10,10,0.08)",
              overflow: "hidden",
              marginBottom: 28,
            }}
          >
            <div
              style={{
                height: "100%",
                width: `${barProgress}%`,
                background: "linear-gradient(90deg, #60a5fa, #a78bfa)",
                borderRadius: 999,
              }}
            />
          </div>

          <div
            style={{
              fontSize: 25,
              lineHeight: 1.55,
              color: "#262626",
              fontWeight: 500,
              minHeight: 150,
            }}
          >
            {visibleText}
            {cursorOn && <span style={{ opacity: 0.6 }}>|</span>}
          </div>
        </GlassPanel>
      </AbsoluteFill>
    </AbsoluteFill>
  );
};
