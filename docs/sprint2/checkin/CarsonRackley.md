# Sprint 2 Mid-Sprint Check-in: Carson Rackley (@CarsonRackley)

## What I have done so far, and what remains

I have not completed much implementation yet. Between coursework for other classes, the in-class
presentations and Quiz 2, the first half of the sprint went mostly to planning. I read through the
whole Sprint 2 page and mapped each deliverable to what I owe: two medium issues, a peer-reviewed
spec and at least one design decision with two alternatives per issue, PRs with a Tech Debt
section, AI logs, and UML diagrams. I then updated our shared Claude context files
(`.claude-students/COURSE-WORKFLOW.md`) with the Sprint 2 rules, formats and deadlines, so anyone on
the team can ask Claude which deliverables are still missing.

What remains is most of the sprint's work. I need to choose my two medium issues and put them on
the Sprint 2 milestone, tighten their descriptions and acceptance criteria, and get each spec
peer-reviewed before I write code. Then I implement both with tests, review a teammate's PR, and
draw the class and sequence diagrams (D6) for one of my issues.

## One thing I learned

The biggest thing I learned is how differently Go organizes code compared to the Java I wrote in
Software Dev. Go has no classes. Instead, data is kept in simple structs, and functions are
attached to them separately. Rather than one class inheriting from another, Go builds bigger pieces
by placing one struct inside another, so it reuses that struct's fields and behavior. Interfaces
also work differently: in Java you have to declare that a class implements an interface, but in Go
anything that has the right functions fits automatically. That is how Gitea can use one shared
database helper to look up many different kinds of data. The web pages are built from templates
that the server fills in with data before sending them to the browser. At first this made the code
hard to follow because I kept looking for classes that weren't there. Once I understood the
pattern, tracing my Sprint 1 notification changes through the code became much easier, and I feel
more confident jumping into unfamiliar parts of the codebase.

## One thing I would do differently

I would start earlier. I let this sprint's work get pushed behind other deadlines, so now most of
it lands in the second week, along with the stress of knowing it is still waiting. Even small steps
in the first few days, like picking my issues and posting their specs, would have helped, because
peer review needs turnaround time that I cannot compress at the end. Going forward I plan to set
aside time early in each sprint just for getting my issues set up.

For the project itself, I would be more exploratory when choosing and sizing issues. Before
committing to one, I want to trace the code it touches first (models, routes, templates and tests)
so I know whether it needs a migration or spans several layers. That gives a more honest estimate,
keeps an issue from turning out bigger than it looked, and means I understand the whole
implementation before I start coding instead of discovering it halfway through.
