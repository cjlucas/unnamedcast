
export function shortDuration(dur) {
  dur = parseInt(dur);
  const units = [
    {amt: 24 * 60 * 60, unit: "d"},
    {amt: 60 * 60, unit: "h"},
    {amt: 60, unit: "m"},
    {amt: 1, unit: "s"},
  ];

  var s = [];

  units.forEach(unit => {
    var n = Math.floor(dur / unit.amt);
    if (n == 0) return;
    dur %= n;
    s.push(`${n}${unit.unit}`);
  });

  return s.join(" ");
}
