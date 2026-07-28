import React from "react";
import { staticFile } from "remotion";
import { Audio } from "@remotion/media";

export const TransitionWhoosh: React.FC = () => {
  return <Audio src={staticFile("sfx/whoosh.wav")} volume={0.18} />;
};
