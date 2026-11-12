 1. What have you implemented so far, and what remains?
    I have implemented 1 of my issues so far. This issue is the first of 7 issues made from the large issue #151. The PR for my issue has been opened and the code 
    review done by Aman is complete. This PR creating a new method, which reads the commented line from the comment's stored diff. There are also a change to the 
    comment template, where a data-suggestion-original attribute was made, along with a new locale key: "Suggested Change." There were also some frontend and styling 
    add ons. I still need to complete my second issue and the mutation testing.

2. What is one thing you learned about testing in your project?
    A testing bug was caught on an AI generated test for my issue that would never fail, which goes against the rule that 
    every test must be failable, otherwise it is not a test. Through this process of understanding the bug,
    I learned that mistakes like this can be generally easy to make when making tests first, then doing the coding after.
    If the code is written in a way where something can break it, like in this situation, the test should account for the
    possibilities. 

3. What is one risk to finishing your work by Nov 19, and what will you do about it?
    As of right now, there are no risks. Especially now that one PR is done and out of the way early, I already
    have my other issue selected and ready to start implementation. The only add-on is going to be doing the mutation
    testing, but using the tools provided, it will not be difficult. My team is good at communicating when
    things are ready to be checked, so I doubt we will run into any issues. 
