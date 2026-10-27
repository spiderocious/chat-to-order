# Go-live note for Iya Bisi

**What it does.** Customers message your WhatsApp number. The assistant shows today's menu from your admin panel, takes the order, sends a Paystack payment link, and confirms the order only after Paystack says the money arrived. Your portion counts go down by themselves. You watch orders arrive on the board at the admin panel, on your phone or laptop.

**What it does not do.** It does not send riders, give refunds, handle complaints about food already delivered, or add dishes by itself. You set the menu, prices and portions each morning in the panel. If a dish runs out after someone has paid, the order turns red as "Needs you" and you call the customer to swap it or refund by transfer yourself.

**Before real customers use it.**
1. Your own Paystack business account with live keys and your settlement bank account. Today it runs on test keys; test cards work, real cards do not.
2. Your WhatsApp Business number connected to Chord. Today it runs in Chord's simulator.
3. Real prices and portions in the panel, and the service windows set to your real hours.
4. A Chord credit top-up and somebody watching the balance. When credits run out the assistant goes silent.
5. A one-week trial with five regulars you trust, with you watching the board.

**Money.** Prices in the panel are what the customer pays, plus Paystack's fee if your Paystack account is set to pass the fee to the customer (today it is, so a ₦8,700 order shows as ₦8,934 on the payment page). Settle that choice in Paystack before launch. The assistant never takes an account number or a transfer screenshot as proof of payment.

**When a payment fails.** The customer sees the bank's reason and gets a fresh link. Nothing is recorded and no food is reserved. If a customer closes the payment page without finishing, the link stays open for 30 minutes, then the order expires on its own.

**When something breaks.** If Paystack or the ordering system is down, the assistant tells the customer nothing was charged and to try again in a minute, and the Activity page shows the error. If the assistant cannot answer at all, customers get no reply; call Feranmi. Alerts to your phone when an order needs you or the service is down are the first thing to add after launch.

**Honest risks.** Anyone who knows your number can order; the order number is the identity. The assistant can misread a message and ask again rather than guess. Two customers paying for the last portion at the same instant: one is confirmed, the other is flagged for you. The panel runs on a single small server; it is fine for one kitchen at today's volume and would need a bigger setup for several.

**Next.** Adding dishes by sending a photo of your handwritten menu, and a WhatsApp alert when an order needs you. Gadget sellers are the next vertical: same assistant, plus stock counts, warranties and pay-on-delivery, which this version does not do.
