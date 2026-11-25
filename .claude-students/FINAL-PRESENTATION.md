# Final project presentation — Dec 1 and 3

Course requirement, not an upstream Gitea one. Source: <https://csci-435-se.github.io/presentations/final/>
(page source: `docs/presentations/final.md` in `CSCI-435-SE/CSCI-435-SE.github.io`). Sprint 4
deliverables are in `COURSE-WORKFLOW.md`; the presentation is graded separately from Sprint 4.

## Logistics

- **When:** Tuesday Dec 1 and Thursday Dec 3, 12:30 PM, in class. Three teams present each day, in
  a random order drawn in class, so **every team must be ready on Dec 1**. Every member attends
  **both** days.
- **Length:** 15 minutes per team, plus 5 minutes of Q&A.
- **Slides:** a PDF uploaded to Blackboard by **12:20 PM on Dec 1**, including teams presenting on
  Dec 3.
- **Points:** 100 (9% of the course grade).
- **Format:** team-level. Any member may present any section, and every member should be ready to
  speak to, and take Q&A on, any section.
- **Content:** the team's **Sprint 3 work** (including the team feature) and its **plan for
  Sprint 4**. Section times below are suggestions; order and timing are the team's choice as long
  as every requirement is covered within 15 minutes.

## Sections

1. **Project and Sprint 3 outcomes (1.5 min).** A short reminder of what the project is and does.
   One slide (a table works well) with issues completed, story points delivered and the percentage
   of the Sprint 3 backlog completed. Below 100%: what wasn't completed and **why**. At 100%: what
   was key to it.
2. **Three main features from Sprint 3 (4 min).** One must be the **team feature** (about 2 min):
   what it does and why it matters to users; what works at the end of Sprint 3, as screenshots of
   the running feature or a short clip (no code screenshots); a simple diagram of the main
   components it touches and how they interact; and what doesn't work yet, honestly. Two other
   Sprint 3 features (about 1 min each): what it does, why it matters, and screenshots or a clip.
   No live demo.
3. **Requirements and decisions (2.5 min)**, about the team feature.
   - **Requirements:** how the spec evolved from the draft ACs and open questions: which open
     questions mattered most and how they were answered (customer decisions, upstream behaviour,
     team decisions), and how the ACs changed as a result (one or two concrete examples).
   - **Breakdown and estimation:** how the team split the feature into sub-issues and ordered
     them; how accurate the estimates were, and what was missed.
   - **Decisions:** the **two most significant** design or architecture decisions. For each: what
     was decided; the alternatives considered (at least one real one); what drove the choice
     (reasoning, constraints, trade-offs); and who made it (one person, or the team after
     discussion). If an AI tool evaluated options, how the team validated its recommendation.
4. **Quality evidence (2.5 min).** Evidence from Sprint 3, not claims:
   - **Testing:** how the team decided the expected values in its tests (the "source of expected
     value" in Test Evidence tables), with one example where it mattered.
   - **Mutation testing:** what the surviving mutants revealed about the tests, with one concrete
     gap found and fixed (the write-ups are in `docs/sprint3/mutation/`).
   - **Code review:** one case where a review caught a real problem (a bug, a missing case, a wrong
     assumption), and how it was resolved.
5. **AI and tools in Sprint 3 (2 min).** How the team used AI (e.g. specs, code, tests, mutation
   testing, reviews); **one case where AI helped a lot** and **one where it failed or misled**, with
   what was learned from each; how the team verified AI output.
6. **Lessons and Sprint 4 plan (2.5 min).** What the team learned in Sprint 3 and what it is
   changing in Sprint 4 as a result (process, planning, AI use, collaboration); the plan to finish
   the team feature by **Dec 15**, with the remaining work and how it is split; the main risk to
   finishing, and what the team will do about it.

## Rubric (100 points)

Copied from the presentation page.

| Criterion | What earns full credit |
| --- | --- |
| **1. Project and Sprint 3 outcomes (10 pts)** | Sprint 3 metrics are accurate and shown on a dedicated, readable slide. Shortfalls are explained honestly. |
| **2. Three main features (20 pts)** | The team feature is explained clearly, with readable visual evidence, a simple diagram of the main components it touches, and an honest account of what does not work yet. Two other Sprint 3 features are explained with visual evidence. |
| **3. Requirements and decisions (15 pts)** | The evolution of the team feature's spec is shown with concrete examples. The breakdown and estimation are explained, including what was missed. Two real decisions are explained with alternatives, rationale, and who decided. |
| **4. Quality evidence (15 pts)** | Testing, mutation testing, and code review are each supported by a concrete example from Sprint 3. Evidence is shown, not claimed. |
| **5. AI and tools (15 pts)** | Sprint 3 AI use is described specifically. One success and one failure are presented with what was learned. Verification practices are addressed. |
| **6. Lessons and Sprint 4 plan (15 pts)** | Lessons are specific to the team's Sprint 3 experience, not generic, and lead to concrete changes. The Sprint 4 plan and main risk are specific and realistic. |
| **7. Delivery (10 pts)** | Presented on time. Slides support the talk; they do not replace it. Any team member can speak to any section. Each section includes a brief, honest reflection on what the team would improve. |

## Slide and delivery guidance

- **Readable from the back of the room:** text, diagram labels, screenshots and video. Crop
  screenshots to the relevant area. Clips stay **under a minute**, zoomed in, narrated live or
  captioned, embedded or ready to play, and tested on the classroom setup beforehand.
- **One message per slide**; a slide that takes more than 5 seconds to read has too much on it.
- Slides carry the evidence; the presenter adds the meaning. Prefer visuals when the evidence is
  visual. Keep fonts, colours and layout consistent across the deck.
- **No member-listing slide:** team name and project go on the title slide; introduce yourselves
  verbally.
- **Works well:** architecture diagrams showing how components connect; before/after UI
  screenshots; a screenshot of an issue showing how its ACs changed; a cropped test or review
  comment that caught a real problem; terminal output of build, test or mutation results instead of
  "our tests pass"; a simple chart or table for the sprint trend.
- **Avoid:** directory trees or code screenshots as slides; listing 10+ issues by name and number
  (synthesise into themes or show the one that matters); vague lines like "it was challenging but
  we got it working" (name the obstacle and what resolved it); architecture slides with no
  diagram; generic lessons or risks ("communication is important", "other coursework").
- **Discouraged:** reading verbatim from paper (brief notes are fine), reading the slides verbatim,
  and listing issues or tasks one by one without synthesis or reflection.
