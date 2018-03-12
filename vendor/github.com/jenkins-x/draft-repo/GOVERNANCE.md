# Project Governance

There are two separate individual ranks that govern the Draft project. These are

1. Admiral
2. Commodore

See [OWNERS](OWNERS) for the list of individuals that govern the Draft project.

## Admiral

Admirals are engineers who either work or contribute to Draft on a frequent basis. They have
significant experience with the project and are considered "ship masters". The Admirals are
responsible for

- Technical direction
- Project governance and process (including this policy)
- Contribution policy
- Conduct guidelines
- Maintaining the list of individuals given the rank of Commmodore.

## Commodore

Individuals making significant and valuable contributions are assigned the rank of Commodore upon
request. These individuals are identified by the Admirals and have moderate or extensive experience
working with the Draft project. Commodores may have specific domain knowledge on a certain package or
contribute often to the project, but may also have certain gaps of knowledge over the project or
contribute on a less frequent schedule to be promoted to Admiral.

Commodores may also review PRs (and are encouraged to do so!), however they should consider if they
understand the domain. If they don't understand a particular piece of code or a specific design
decision, they should reach out and ask for clarification from the submitter or from an Admiral.

# LGTM Policy

When reviewing pull requests, Draft uses a LGTM (Looks Good To Me!) policy. Because of the velocity
of the project in its given state (pre-v1.0), the LGTM policy is as follows:

## Pull Requests Submitted by Admirals

Small PRs submitted by an Admiral only requires a single LGTM from another Admiral or a Commodore.
This is because an Admiral is identified as an individual with significant experience with the
project, so it is assumed that smaller features have already been "signed off".

Larger PRs that alter behaviour significantly from what's in master needs to be signed off by two
Admirals or Commodores, but only one of them needs to review it. This is to ensure a proper transfer
of knowledge is passed on to other Admirals and Commodores, reducing overall
[bus factor](https://en.wikipedia.org/wiki/Bus_factor), while still ensuring the project can
continue at its current velocity.

The sign-off process is completely informal. A "full steam ahead!" on Slack is more than acceptable.

Scenario: there are two Admirals and a Commodore. Admiral "a" proposes a certain feature that alters
how Draft operates in a significant way. Admiral "b" and the Commodore both approve the proposal
(informally), and Admiral "b" reviews the pull request.

## Pull Requests Submitted by Commodores

The same policy applied to Admirals also applies to Commodores. Commodores are seen in the same
light as Admirals when it comes to code contributions; they just have less overall responsibility to
maintain the project's direction and governance.

## Pull Requests Submitted by the Community

All PRs, small or big, need to be signed off by two Admirals/Commodores and reviewed by one.
