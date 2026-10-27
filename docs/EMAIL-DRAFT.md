# Reply to "Next steps with TSG Labs: your take-home assignment"

Subject: Re: Next steps with TSG Labs: your take-home assignment

Hi Emmanuel,

**Role: Forward-deployed Engineering** (second choice: Technical Product Management).

Here is my submission: [Google Drive folder link]

**Problem.** Iya Bisi runs a home kitchen in Surulere and sells lunch and dinner to offices and homes nearby over WhatsApp. She answers the same three questions all day, takes orders by hand, matches bank transfers to names from screenshots, and oversells dishes because the portion count lives in her head. The agent shows today's menu, takes the order, collects payment through a Paystack link, and records the paid order with stock reduced, so she cooks instead of typing and never confirms an order that was not paid. It deliberately leaves out delivery dispatch, refunds, multiple vendors and building the menu from photos.

**The working agent.** Teammate "Iya Bisi's Kitchen" is live on my Chord account and reachable in the WhatsApp simulator. The vendor's admin panel is at https://chat-to-order-986.netlify.app and the tool server at https://backend-production-c54f.up.railway.app. The tools call Paystack in test mode; the success test option on the checkout page completes a payment.

**What is in the folder.**
1. Problem statement and links
2. The 3-minute recording
3. Deployment field notes: eight real problems, with the error text, how I found each cause, the fix, and what I would change in Chord, Paystack or Railway
4. Go-live note for the business owner
5. Code: https://github.com/spiderocious/chat-to-order (Go + Gin MCP server and API, React admin panel, the skill files, tests, a 50-check end-to-end smoke script, and the technical designs)

Credits used: under 1 of 50, so there is plenty left for your own test.

Thank you for the exercise. Asking about the built-in connector early was useful; I built my own MCP server because the integration and the field notes are where I learned the most.

Best regards,
Feranmi Adeniji
