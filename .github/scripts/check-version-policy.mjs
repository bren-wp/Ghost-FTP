import assert from "node:assert/strict";
import { validateVersionTrain } from "./version-policy.mjs";

for (const [previous, next] of [
  ["0.30.21", "0.90.0"],
  ["0.90.0", "0.90.1"],
  ["0.90.1", "0.90.2"],
  ["0.90.1", "0.91.0"],
  ["0.98.4", "0.99.0"],
  ["0.99.3", "1.0.0"],
  ["1.0.0", "1.0.1"],
  ["1.0.0", "1.1.0"],
  ["1.1.0", "1.1.1"],
  ["1.1.1", "1.1.2"],
  ["0.99.99", "1.0.0"],
  ["1.1.1", "1.2.0"],
]) assert.doesNotThrow(() => validateVersionTrain(previous, next), `${previous} -> ${next}`);

for (const [previous, next] of [
  ["0.30.21", "0.30.22"],
  ["0.30.21", "0.91.0"],
  ["0.90.0", "0.90.0"],
  ["0.90.0", "0.90.2"],
  ["0.90.1", "0.90.1"],
  ["0.99.0", "0.100.0"],
  ["0.90.0", "0.92.0"],
  ["0.90.1", "0.91.1"],
  ["0.98.0", "1.0.0"],
  ["0.99.2", "1.1.0"],
  ["1.0.0", "1.2.0"],
  ["1.1.1", "1.1.0"],
  ["1.0.0", "0.99.0"],
]) assert.throws(() => validateVersionTrain(previous, next), `${previous} -> ${next}`);

console.log("Ghost FTP version train policy regression matrix passed.");
