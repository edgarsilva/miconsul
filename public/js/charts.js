// Renders the daisyUI ApexCharts charts on a page. A chart is an element with
// data-chart="<kind>" and its data as JSON in data-chart-data. ApexCharts and
// its stylesheet load once, from jsdelivr with SRI, the first time a page has a
// chart. Colors are theme variables, so charts follow theme changes.
(() => {
  const apexVersion = "6.10.0";
  const apexScript = {
    src: `https://cdn.jsdelivr.net/npm/apexcharts@${apexVersion}/dist/apexcharts.min.js`,
    integrity: "sha384-5EG9CeSqSzymwY307XCL4zMqifjcpwh/uGHYg1CPiyMNHVvfCcx8VZi2JzW2035O",
  };
  const apexStyles = {
    href: `https://cdn.jsdelivr.net/npm/apexcharts@${apexVersion}/dist/apexcharts.css`,
    integrity: "sha384-xiwYvoUk6woTSrViYmFRB6voHXkdMsbOIPNggFn88nizTDFhys4ThD1QO/AhVyJA",
  };

  const baseContent = "var(--color-base-content)";
  const shade = (transparent) => `color-mix(in srgb, var(--color-base-content), transparent ${transparent}%)`;
  // Tooltip dots use classes, not inline styles, which the CSP blocks.
  const shades = [
    { opacity: 1, dot: "bg-base-content" },
    { opacity: 0.7, dot: "bg-base-content/70" },
    { opacity: 0.5, dot: "bg-base-content/50" },
    { opacity: 0.3, dot: "bg-base-content/30" },
  ];

  const escapeHTML = (value) =>
    String(value).replace(/[&<>"']/g, (ch) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[ch]);

  const tooltip = (dot, label, value) => `
    <div class="bg-base-100 border-(length:--border) border-base-content/10 rounded-box p-3 text-xs">
      <div class="flex items-center gap-2">
        <span class="size-2 rounded-selector ${dot}"></span>
        <span class="text-base-content/70">${escapeHTML(label)}: <b class="text-base-content ml-1">${escapeHTML(value)}</b></span>
      </div>
    </div>`;

  // Column Chart One: monthly revenue columns.
  const columns = (data) => ({
    chart: { height: "300px", type: "bar", fontFamily: "inherit", toolbar: { show: false }, injectStyleSheet: false },
    series: [{ name: data.name, data: data.values }],
    colors: [baseContent],
    plotOptions: { bar: { horizontal: false, columnWidth: "55%", borderRadius: 5, borderRadiusApplication: "end" } },
    dataLabels: { enabled: false },
    stroke: { show: true, width: 2, colors: ["transparent"] },
    xaxis: {
      categories: data.labels,
      labels: { style: { colors: baseContent, fontSize: "12px" } },
      axisBorder: { show: true, color: shade(90) },
      axisTicks: { show: true, color: shade(90) },
    },
    yaxis: { show: false },
    fill: { opacity: 1 },
    grid: { borderColor: shade(95) },
    legend: { show: false },
    tooltip: {
      enabled: true,
      custom: ({ dataPointIndex }) => tooltip(shades[0].dot, data.titles[dataPointIndex], data.formatted[dataPointIndex]),
    },
  });

  // Pie Chart Nine: share donut with the total in the middle.
  const donut = (data) => ({
    chart: { height: "220px", type: "donut", fontFamily: "inherit", toolbar: { show: false }, injectStyleSheet: false },
    series: data.values,
    labels: data.labels,
    colors: data.values.map(() => baseContent),
    fill: { opacity: data.values.map((_, i) => shades[i % shades.length].opacity) },
    stroke: { show: true, width: 1, colors: ["var(--color-base-200)"] },
    plotOptions: {
      pie: {
        expandOnClick: false,
        donut: {
          size: "62%",
          labels: {
            show: true,
            name: { show: true, fontSize: "11px", color: baseContent, offsetY: -2 },
            value: {
              show: true,
              fontSize: "18px",
              fontWeight: 700,
              color: baseContent,
              offsetY: 4,
              formatter: (_, opts) => data.formatted[opts.seriesIndex] ?? "",
            },
            total: { show: true, showAlways: true, label: data.totalLabel, fontSize: "11px", color: baseContent, formatter: () => data.total },
          },
        },
      },
    },
    dataLabels: { enabled: false },
    legend: { show: false },
    tooltip: {
      enabled: true,
      custom: ({ seriesIndex }) =>
        tooltip(shades[seriesIndex % shades.length].dot, data.labels[seriesIndex], data.formatted[seriesIndex]),
    },
  });

  const kinds = { columns, donut };

  const loadApex = () => {
    if (window.ApexCharts) return Promise.resolve();
    if (!document.querySelector(`link[href="${apexStyles.href}"]`)) {
      const link = document.createElement("link");
      link.rel = "stylesheet";
      link.href = apexStyles.href;
      link.integrity = apexStyles.integrity;
      link.crossOrigin = "anonymous";
      document.head.append(link);
    }
    let script = document.querySelector(`script[src="${apexScript.src}"]`);
    if (!script) {
      script = document.createElement("script");
      script.src = apexScript.src;
      script.integrity = apexScript.integrity;
      script.crossOrigin = "anonymous";
      document.head.append(script);
    }
    return new Promise((resolve, reject) => {
      script.addEventListener("load", resolve, { once: true });
      script.addEventListener("error", reject, { once: true });
    });
  };

  const render = async () => {
    const elements = [...document.querySelectorAll("[data-chart]:not([data-chart-rendered])")];
    if (elements.length === 0) return;
    await loadApex();
    for (const el of elements) {
      const build = kinds[el.dataset.chart];
      if (!build) continue;
      el.dataset.chartRendered = "true";
      new window.ApexCharts(el, build(JSON.parse(el.dataset.chartData))).render();
    }
  };

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", render, { once: true });
  } else {
    render();
  }
})();
