import React from "react";
import { Img, staticFile } from "remotion";

export const LogoMark: React.FC<{ size?: number; style?: React.CSSProperties }> = ({
  size = 160,
  style,
}) => {
  return (
    <Img
      src={staticFile("logo-mark.svg")}
      style={{ width: size, height: (size * 615) / 732, ...style }}
    />
  );
};
