// Gives the seller role to every account that already owns a shop.
//
// Opening a shop now requires the seller role, which an admin grants. The
// accounts that opened one before that rule existed all still read role="user",
// so without this they would lose the seller side of the app overnight.
//
// Idempotent: accounts that are already seller or admin are left alone.
//
//   docker exec -i vngrocery-mongo mongosh --quiet \
//     < server/scripts/backfill_seller_role.js
//
// or, from a shell that can reach mongo directly:
//
//   mongosh "mongodb://127.0.0.1:27017/vngrocery" scripts/backfill_seller_role.js

const database = db.getSiblingDB("vngrocery");

const owners = database.shops.distinct("ownerUserId", {
  status: { $ne: "deleted" },
  ownerUserId: { $nin: [null, ""] },
});

if (owners.length === 0) {
  print("no shop owners found; nothing to do");
} else {
  const pending = database.users
    .find(
      { userId: { $in: owners }, role: { $nin: ["seller", "admin"] } },
      { userId: 1, email: 1, role: 1 },
    )
    .toArray();

  print(`shop owners: ${owners.length}`);
  print(`to promote:  ${pending.length}`);

  for (const user of pending) {
    // version is the optimistic-concurrency counter the API checks on every
    // write, so a backfill has to move it like any other update would.
    const result = database.users.updateOne(
      { userId: user.userId },
      {
        $set: { role: "seller", updatedAt: new Date() },
        $inc: { version: 1 },
      },
    );
    print(
      `  ${user.email ?? user.userId}: ${user.role} -> seller` +
        (result.modifiedCount === 1 ? "" : " (FAILED)"),
    );
  }

  const left = database.users.countDocuments({
    userId: { $in: owners },
    role: { $nin: ["seller", "admin"] },
  });
  print(
    left === 0 ? "done: every shop owner is a seller" : `still not seller: ${left}`,
  );
}
