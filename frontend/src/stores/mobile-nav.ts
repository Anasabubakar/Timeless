import { create } from "zustand";

interface MobileNavState {
  open: boolean;
  setOpen: (open: boolean) => void;
}

// Lets the topbar's mobile search icon and the dock's center button open
// the same full-screen nav dialog without prop-drilling through layout.tsx.
export const useMobileNavStore = create<MobileNavState>((set) => ({
  open: false,
  setOpen: (open) => set({ open }),
}));
