# Mid-Sprint Check-in
## Aman Bhoot (ab-wm)

### Question 1: What have you implemented so far, and what remains?
All of my issues that I planned for, that is, #143, #144, and #145, has been completed, PRs made, and merged. This includes the mutation testing report as well. The issues represent two medium issues of 3 points each, but to be sure that I implemented enough work, I also completed the small issue. Other than that, I have also already reviewed two PRs.

### Question 2: What is one thing you learned abouut testing in your project?
Being able to isolate changes of the issue with code that already exists is quite important in order to both run mutation tests properly, since every additional code path means even more tests, but also to narrow down the generated report. In my case, most found failing mutations were from code that already existed and unmodified by me (184 surviving mutants!). Any future mutation testing will similarly carry the same problems, and so mutation tests need to be smartly done.

There is also the aspect that to actually judge a mutation is quite tricky, and requires a fair bit of knowledge of the introduced changes to accurately judge if the mutants are even worth addressing. For example, a couple of the surviving mutants could not even be accounted for, as that would involve guessing or controlling external factors that, reasonably, should not be the concern of the code that was being changed.

### Question 3: What is one risk to finishing your work by Nov 19, and what will you do about it?
As mentioned in question 1, I have completed everything I need to do for this sprint, I believe. I should clarify this is a result of how I work, where I dedicate all my time to just one assingment at a time as most of my assignments are multi-week projects. Thus, I spent the entirety of a few days getting all the work done.
