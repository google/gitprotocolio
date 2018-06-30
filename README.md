# gitprotocolio

A Git protocol parser written in Go.

This is more like an experimental project to better understand the protocol.

This is not an official Google product (i.e. a 20% project).

## Background

Git protocol is defined in
[Documentation/technical/pack-protocol.txt](https://git.kernel.org/pub/scm/git/git.git/tree/Documentation/technical/pack-protocol.txt).
This is not a complete definition. Also, a transport specific spec
[Documentation/technical/http-protocol.txt](https://git.kernel.org/pub/scm/git/git.git/tree/Documentation/technical/http-protocol.txt)
is not complete. This project was started so that these upstream protocol spec
becomes more accurate. To verify the written syntax is accurate, this project
includes a Git protocol parser written in Go, and have end-to-end test suites.

This makes it easy to write a test case for Git client. Currently the test cases
are run against the canonical Git implementation, but this can be extended to
run against JGit, etc.. Also it makes it easy to test an attack case. With this
library, one can write an attack case like
[git-bomb](https://github.com/Katee/git-bomb) against Git protocol by producing
a request that is usually not produced by a sane Git client. Protocol properties
can also be checked. For example, it's possible to write a test to check valid
request/response's prefixes are not a valid request/response. This property
makes sure a client won't process an incomplete response thinking it's complete.

## TODOs

*    Protocol semantics is not defined.

     The syntax is relatively complete. The semantics is not even mentioned. One
     idea is to define the semantics by treating the request/response as an
     operation to modify Git repositories. This perspective makes it possible to
     define a formal protocol semantics in a same way as programming language
     formal semantics.

     Defining a simple git-push semantics seems easy. Defining a pack
     negotiation semantics for shallow cloned repositories seems difficult.

*    Upstream pack-protocol.txt is not updated.

     The initial purpose, create a complete pack-protocol.txt, is not yet done.
     We can start from a minor fix (e.g. capability separator in some places is
     space not NUL). Also relationship between Git protocol and network
     transports (HTTPS, SSH, Git wire) are good to be mentioned.

*    Bidi-transports are not tested and defined.

     Git's bidi-transports, SSH and Git-wire protocol, are not tested with this
     project and the protocol syntax is also not defined. The majority of the
     syntax is same, but there's a slight difference. Go has an SSH library, so
     it's easy to run a test SSH server.
