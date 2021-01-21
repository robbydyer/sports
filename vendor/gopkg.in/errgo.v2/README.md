Package errgo provides some primitives for error creation and handling.

It provides primitives for wrapping and annotating errors without exposing
implementation details unnecessarily.

Error dependencies as code fragility

If we import a package, how much of its exposed API are we entitled
to use? Most packages do not explicitly document the set of possible
public errors that can be returned from their functions, so if we see
that a package returns a particular type or value, can we be blamed for
writing code that relies on it?

I cannot be the only one that has run a function that's returning an error
I want to handle, looked at the error's type and added a test against
that type.  That type isn't necessarily in the package that I'm calling,
but may be several layers deeper than that.  Once that condition is there
in the my code, it may be broken if any of the layers below it fails to
preserve or generate the errors values my code is now expecting.

For example, say I'm developing text-editor integration for my favourite
cloud app. When using it to parse the app's configuration data, I find
that it returns json.SyntaxError when there's a configuration error. I
write code that, when it finds json.SyntaxError, jumps the editor cursor
to the place where the error was found (by looking at SyntaxError.Offset).

Some time later, someone comes along and decides that encoding/json is
too slow, and replaces it with xxx/fastjson.  Since encoding/json isn't
in the public API surface area, this doesn't seem like a breaking change,
so the change lands, but in actual fact my code is now broken.

When code bases get sufficiently large, this makes refactoring code
considerably more error-prone, because it is not always
clear whether it's OK to change some code or not. The code
is fragile.

Wrapping breaks error dependencies

A common idiom is to return an error "annotated" with some extra text
to give it more context. If we do that, then we protect ourselves
from the subtle breakage illustrated above, because callers *cannot*
depend on the underlying error any more. If I want to add my feature,
I need to notify upstream and tell them that I'd like to be able to
find where the config file syntax error is, and they'll need to land a
change that makes it explicitly available in the API. This is more work,
but it means that the error type is now an explicit part of the API
and hence much less likely to be changed in a backwardly incompatible
way.

The problem with fmt.Errorf is that it hides even error causes that
we *want* to expose. It's common to have a few well-known error
return values

I find a package
that can talk to the app, and I find that when


My code uses
the tool's config p


calling a package that uses (internally)
encoding/json to parse its configuration format. I'm building
an 



Even if we're only importing
a package for internal use, if we're returning its errors,
we're exposing ourselves to that kind of 

 example, changing to use
a different package that's only used internally
might end up breaking some other code that happens
to be 
if any of those
layers changes to use a different lower layer,
or 

Fragility 1: "That code doesn't return that kind of error any more!"

Fragility 2: "I thought that error meant *this* not *that* as well!"


Some code makes a conditional test on an error.  It may compare the
error for equality with a value defined in another package (io.EOF), or
call a function (os.IsNotExist) or do a dynamic type comparison. There
is a dependency made between the code doing the comparison and the code
that defines the value, function or type. This dependency is not visible
directly in any call graph but bugs and API incompatibilities can easily
arise when dependencies change.

Since errors can propagate back through many layers of stack and layers
of abstraction (and are commonly returned exactly as received), a caller
might be testing an error returned from a package that an intermediate
package considers an implementation details.

Even though error values and types are not generally well documented
(or perhaps *because* they're not), these dependencies arise commonly
because, on finding some error that we want to handle, we will run the
program or look at the code, then write a test against the type we see.

This 


and try to find something about the error that can be used to
distinguish the problem we're seeing, and often that will involve
a test against an error␣ value 

I believe that most programmers
will try to avoid string comparisons (known to be fragile), but testing
against a known type seems more robust.


 diagnoses
the problem what we're seeing.

some kind of database
package, and as part of that implementation, it uses an
encoding package.

initially uses a file based store.

It is currently very common to return exactly the error that was


when encountering an error


When code tests an error against a particular type or value
to decide what to do


The  is informed by the view that 

hiding is important



 It aims to decrease implicit inter-package dependency