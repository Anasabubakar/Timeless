import React from "react";
import { TransitionSeries, linearTiming } from "@remotion/transitions";
import { fade } from "@remotion/transitions/fade";
import { LogoIntro } from "./scenes/LogoIntro";
import { Tagline } from "./scenes/Tagline";
import { PipelineFeature } from "./scenes/PipelineFeature";
import { AgentsFeature } from "./scenes/AgentsFeature";
import { ProposalFeature } from "./scenes/ProposalFeature";
import { AnalyticsFeature } from "./scenes/AnalyticsFeature";
import { Outro } from "./scenes/Outro";

export const SCENE_DURATIONS = {
  logoIntro: 100,
  tagline: 110,
  pipeline: 170,
  agents: 170,
  proposal: 170,
  analytics: 160,
  outro: 130,
};

export const TRANSITION_DURATION = 20;

const TRANSITION_COUNT = 6;
export const TOTAL_DURATION =
  Object.values(SCENE_DURATIONS).reduce((a, b) => a + b, 0) -
  TRANSITION_COUNT * TRANSITION_DURATION;

const Crossfade: React.FC = () => (
  <TransitionSeries.Transition
    presentation={fade()}
    timing={linearTiming({ durationInFrames: TRANSITION_DURATION })}
  />
);

export const LaunchVideo: React.FC = () => {
  return (
    <TransitionSeries>
      <TransitionSeries.Sequence durationInFrames={SCENE_DURATIONS.logoIntro}>
        <LogoIntro />
      </TransitionSeries.Sequence>
      <Crossfade />
      <TransitionSeries.Sequence durationInFrames={SCENE_DURATIONS.tagline}>
        <Tagline />
      </TransitionSeries.Sequence>
      <Crossfade />
      <TransitionSeries.Sequence durationInFrames={SCENE_DURATIONS.pipeline}>
        <PipelineFeature />
      </TransitionSeries.Sequence>
      <Crossfade />
      <TransitionSeries.Sequence durationInFrames={SCENE_DURATIONS.agents}>
        <AgentsFeature />
      </TransitionSeries.Sequence>
      <Crossfade />
      <TransitionSeries.Sequence durationInFrames={SCENE_DURATIONS.proposal}>
        <ProposalFeature />
      </TransitionSeries.Sequence>
      <Crossfade />
      <TransitionSeries.Sequence durationInFrames={SCENE_DURATIONS.analytics}>
        <AnalyticsFeature />
      </TransitionSeries.Sequence>
      <Crossfade />
      <TransitionSeries.Sequence durationInFrames={SCENE_DURATIONS.outro}>
        <Outro />
      </TransitionSeries.Sequence>
    </TransitionSeries>
  );
};
