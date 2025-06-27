![lightning logo](./logo.svg)

# lightning

- **Connecting Communities**: bridges many popular messaging apps
- **Extensible**: support for messaging apps provided by plugins which can be
  enabled/disabled by the user
- **Easy to run**: able to run in Docker with multiple database options
- **Based on Go**: uses the strong typing, performance and simplicity of Go

## documentation

- [_User Guide_](https://williamhorning.eu.org/lightning/users)
- [_Hosting Docs_](https://williamhorning.eu.org/lightning/hosting)
- [_Development Docs_](https://williamhorning.eu.org/lightning/developer)

## the problem - and solution

If you've ever had a community, chances are you talk to them in many different
places. Over time, you end up with fragmentation as your community starts to
grow. Many people end up using multiple apps for just your community, people get
upset about the differences between apps, and it becomes a mess.

Now, you could just say "_X is the only chat app we're using from now on_", but
that risks alienating your community.

What other options are there? Bridging! Everyone gets to use their preferred
app, gets the same messages, and is on the same page.

## prior art

Many other bridges exist, however, many of them have issues. Some bridges didn't
play well with others, others didn't handle attachments, others refused to
handle embedded media, and it was a mess. With lightning, I wanted to solve
these issues by bringing many platforms with one tool, having it handle
everything.

## supported platforms

Currently, Discord, Guilded, Revolt, and Telegram are supported. Support for
more platforms is possible to do, but support for these platforms should be
similar to other supported platforms and messages should be presented as
similarly to other messages as possible.

### requesting another platform

If you would like support for another platform, please open an issue! I'd love
to add support for more platforms, though there are a few requirements they
should fulfil:

1. having a pre-existing substantial user base
2. having Go libraries with decent code quality
3. having rich-messaging support of some kind

### matrix notes

The Matrix Specification is really difficult to correctly handle. Solutions that
_work well_ aren't easy to implement, and I've tried implementing support via an
appservice, but, with MSC4144, I'm hoping that this will become easier to do
