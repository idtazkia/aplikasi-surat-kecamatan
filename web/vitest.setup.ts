// Setup file untuk vitest — explicit happy-dom Window registration kalau
// auto-environment tidak bekerja.
import { GlobalRegistrator } from "@happy-dom/global-registrator";

GlobalRegistrator.register();
