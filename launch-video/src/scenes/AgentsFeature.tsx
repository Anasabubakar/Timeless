import React from "react";
import { AbsoluteFill, useCurrentFrame, interpolate, Easing } from "remotion";
import { Background } from "../components/Background";
import { LogoMark } from "../components/Logo";
import { TransitionWhoosh } from "../components/TransitionWhoosh";
import { interFontFamily } from "../fonts";

const AGENTS = [
  { label: "Research", color: "#60a5fa" },
  { label: "Qualify", color: "#22c55e" },
  { label: "Company Intel", color: "#a78bfa" },
  { label: "Outreach", color: "#f59e0b" },
  { label: "Proposal", color: "#f472b6" },
];

const BOX = 860;
const CENTER = BOX / 2;
const ORBIT_RADIUS = 350;

export const AgentsFeature: React.FC = () => {
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

  const coreP = interpolate(frame, [8, 30], [0, 1], {
    easing: Easing.bezier(0.34, 1.3, 0.64, 1),
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  const rotation = interpolate(frame, [0, 170], [-14, 26], {
    easing: Easing.inOut(Easing.sin),
  });

  return (
    <AbsoluteFill style={{ fontFamily: interFontFamily }}>
      <TransitionWhoosh />
      <Background
        blobs={[
          { color: "#d6e6ff", size: 900, top: "-18%", left: "60%", driftX: -25, driftY: 20 },
          { color: "#e3d6ff", size: 780, top: "56%", left: "-14%", driftX: 20, driftY: -20 },
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
            marginBottom: 8,
          }}
        >
          <div style={{ fontSize: 58, fontWeight: 700, color: "#0a0a0a", letterSpacing: "-0.01em" }}>
            A team of specialists, on call
          </div>
          <div style={{ fontSize: 26, fontWeight: 500, color: "#737373", marginTop: 10 }}>
            Five AI agents working your pipeline in parallel
          </div>
        </div>

        <div style={{ position: "relative", width: BOX, height: BOX }}>
          <svg
            width={BOX}
            height={BOX}
            style={{ position: "absolute", top: 0, left: 0 }}
          >
            {AGENTS.map((agent, i) => {
              const angle = (rotation + (360 / AGENTS.length) * i) * (Math.PI / 180);
              const x = CENTER + Math.cos(angle) * ORBIT_RADIUS;
              const y = CENTER + Math.sin(angle) * ORBIT_RADIUS;
              const lineP = interpolate(frame, [20 + i * 8, 40 + i * 8], [0, 1], {
                easing: Easing.bezier(0.16, 1, 0.3, 1),
                extrapolateLeft: "clamp",
                extrapolateRight: "clamp",
              });
              return (
                <line
                  key={agent.label}
                  x1={CENTER}
                  y1={CENTER}
                  x2={CENTER + (x - CENTER) * lineP}
                  y2={CENTER + (y - CENTER) * lineP}
                  stroke="rgba(10,10,10,0.16)"
                  strokeWidth={2}
                  strokeDasharray="1 7"
                  strokeLinecap="round"
                />
              );
            })}
          </svg>

          <div
            style={{
              position: "absolute",
              top: CENTER,
              left: CENTER,
              translate: "-50% -50%",
              scale: coreP,
              opacity: coreP,
              width: 168,
              height: 168,
              borderRadius: "50%",
              background: "rgba(255,255,255,0.7)",
              backdropFilter: "blur(20px) saturate(180%)",
              border: "1px solid rgba(255,255,255,0.8)",
              boxShadow: "0 24px 60px rgba(15,15,20,0.14), inset 0 1px 0 rgba(255,255,255,0.8)",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <LogoMark size={78} />
          </div>

          {AGENTS.map((agent, i) => {
            const angle = (rotation + (360 / AGENTS.length) * i) * (Math.PI / 180);
            const x = CENTER + Math.cos(angle) * ORBIT_RADIUS;
            const y = CENTER + Math.sin(angle) * ORBIT_RADIUS;
            const p = interpolate(frame, [28 + i * 8, 48 + i * 8], [0, 1], {
              easing: Easing.bezier(0.34, 1.3, 0.64, 1),
              extrapolateLeft: "clamp",
              extrapolateRight: "clamp",
            });
            return (
              <div
                key={agent.label}
                style={{
                  position: "absolute",
                  top: y,
                  left: x,
                  translate: "-50% -50%",
                  opacity: p,
                  scale: p,
                  display: "flex",
                  alignItems: "center",
                  gap: 10,
                  background: "rgba(255,255,255,0.72)",
                  backdropFilter: "blur(16px) saturate(180%)",
                  border: "1px solid rgba(255,255,255,0.75)",
                  borderRadius: 999,
                  padding: "12px 22px",
                  boxShadow: "0 14px 34px rgba(15,15,20,0.08)",
                  whiteSpace: "nowrap",
                }}
              >
                <div
                  style={{
                    width: 10,
                    height: 10,
                    borderRadius: "50%",
                    background: agent.color,
                  }}
                />
                <div style={{ fontSize: 20, fontWeight: 600, color: "#0a0a0a" }}>
                  {agent.label}
                </div>
              </div>
            );
          })}
        </div>
      </AbsoluteFill>
    </AbsoluteFill>
  );
};
