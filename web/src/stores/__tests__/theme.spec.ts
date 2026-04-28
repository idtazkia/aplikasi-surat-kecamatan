import { describe, it, expect, beforeEach } from "vitest";
import { setActivePinia, createPinia } from "pinia";
import { useThemeStore } from "../theme";

describe("theme store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    localStorage.clear();
  });

  it("default light mode kalau localStorage kosong", () => {
    const theme = useThemeStore();
    expect(theme.dark).toBe(false);
  });

  it("baca dark dari localStorage", () => {
    localStorage.setItem("surat-kec-theme", "dark");
    setActivePinia(createPinia());
    const theme = useThemeStore();
    expect(theme.dark).toBe(true);
  });

  it("toggle membalik state + persist", () => {
    const theme = useThemeStore();
    expect(theme.dark).toBe(false);

    theme.toggle();
    expect(theme.dark).toBe(true);
    expect(localStorage.getItem("surat-kec-theme")).toBe("dark");

    theme.toggle();
    expect(theme.dark).toBe(false);
    expect(localStorage.getItem("surat-kec-theme")).toBe("light");
  });
});
