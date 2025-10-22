# lightning: *truly powerful* cross-platform bots

![lightning logo](./logo.svg)

- [the bridge](https://williamhorning.dev/lightning/bridge) - connect
Discord, Guilded, Stoat, and Telegram
- [the framework](https://williamhorning.dev/lightning/framework) - build
your own cross-platform bots

Lightning is a project developing *truly powerful* cross-platform bots, with
the underlying *Lightning framework* being used for *Lightning bridge*,
which is what runs *Bolt*, the hosted bridge bot. The goal is to also make the
framework itself usable by other developers, to create their own bots, and to
make the bridge easy to self-host, while also supporting the principles of:

- **Connecting communities**: the Lightning bridge connects Discord, Guilded,
Stoat, and Telegram, allowing communities to connect, wherever they are
- **Extensibility**: the Lightning framework uses plugins to make it easy to
add new features, while keeping the core simple. The bridge is also designed to
be flexible, with options to disable pings, setup subscribe-only channels, and
more.
- **Ease of use**: the Lightning framework is designed to be easy to use,
with a simple API, and the bridge is designed to be easy to set up and use,
with easy-to-understand documentation.
- **Strength**: Lightning is built on Go, making it easy to build, run, and
configure, all while being performant and reliable.

## the bridge bot

If you've ever had a community, chances are you talk to them in many places,
whether that's Discord, Guilded, Stoat, or Telegram. Over time, you end up
with fragmentation as your community grows and changes, with many people using
multiple messaging apps. People eventually grow tired of the differences
between apps, and switching between them, with things becoming a mess.

You could *try* to move everyone to one app, but that might alienate people, so
what do you do, what options do you have?

**Bridging!** Everyone can use their favorite app, gets the same messages, and
is on the same page. Lightning is an easy to use bridge bot that supports
Discord, Guilded, Stoat, and Telegram. To get started, check out the
[getting started guide](https://williamhorning.dev/lightning/bridge/users),
which will walk you through using Bolt, the hosted version of the Lightning bot.
If you want to self-host, read the
[self-hosting guide](https://williamhorning.dev/lightning/bridge/hosting) to
get started.

## the framework

Lightning is a framework for building cross-platform bots, allowing you to make
bots that support multiple platforms without having to worry about
platform-specific code. The framework is built in Go, making it easy to work
with, and is designed to handle things like commands, events, rate-limits,
attachments, and more, all while being battle-tested in Bolt, which has handled
over half-a-million messages during just the summer of 2025.

The framework consists of the core library, which is platform-agnostic, and
plugins, which add support for specific platforms, such as Discord, Guilded,
Stoat, and Telegram. The only platform-specific code is in the plugins, making
it possible to support new platforms without modifying your bot's core logic.

To see a simple example of how to use the framework, check out the
[framework: hello world](https://williamhorning.dev/lightning/framework/hello-world)
guide, which will walk you through creating a simple bot that responds to
messages and commands. For the full documentation, check out the
[framework documentation](https://williamhorning.dev/lightning/framework).

## what's the difference between Bolt and Lightning?

**Lightning** is both the open-source framework and bridge bot that are used to
run **Bolt**, the hosted version of the bridge bot. Bolt is a specific instance
of the Lightning bridge bot, which is hosted by
[William Horning](https://williamhorning.dev/) and is free to use. You can
also self-host your own instance of the Lightning bridge

## what about matrix?

The [Matrix protocol](https://matrix.org) is great, and adding support for it
to the Lightning framework would be amazing, but non-trivial. Implementing all
the features necessary to make a smooth and reliable experience, on par with
other platforms, is something that is being worked on in
[#98](https://github.com/williamhorning/lightning/pull/98), but isn't complete.
[MSC4144](https://github.com/matrix-org/matrix-spec-proposals/pull/4144) is a
new proposal which makes things easier for the bridge use-case, but that still
needs a lot of other work to happen first. Please feel free to contribute if
you'd like!

## licensing

Lightning, the framework and bridge bot, is licensed under the MIT license. The
framework and plugins will always remain under the MIT license, though the
bridge bot may have a different license in the future, but will always be free
to use. Bolt is also free to use, but is also subject to the
[terms of service](https://williamhorning.dev/bolt/legal).
