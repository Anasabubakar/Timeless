import { loadFont as loadInter } from "@remotion/google-fonts/Inter";

export const { fontFamily: interFontFamily } = loadInter("normal", {
  weights: ["400", "500", "600", "700", "800"],
  subsets: ["latin"],
});
