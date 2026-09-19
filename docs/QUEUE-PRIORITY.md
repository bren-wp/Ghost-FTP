# Queue priority

Ghost FTP **0.0.8** supports queue ordering on the maintained Windows and Linux desktop applications. Android exposes transfer lifecycle state with mobile-appropriate controls.

## Desktop behavior

Priority changes are limited to queued work. They must not rewrite:

- transfer identity;
- connection ownership;
- running job state;
- completed/failed terminal state.

Supported desktop operations include:

- move to top;
- move up;
- move down;
- move to bottom;
- pause / resume where supported;
- retry failed jobs;
- cancel;
- clear completed.

The normal Files workspace keeps queue detail readable without exposing unnecessary technical controls when width is constrained.

## Android

Android shows the transfer queue with status, progress and retry state while keeping touch targets readable.

## Runtime evidence

Windows, Linux and Android runtime evidence must be captured from the exact tested source SHA.

## Release boundary

The active native applications are Windows, Linux and Android. The active release shape is **13 platform artifacts / 16 public files**.
