
  const canvas = document.getElementById("canvas");
  const ctx = canvas.getContext("2d");

  async function loadAndRender(x, y) {
    const url = new URL("http://localhost:8080/in");
    url.searchParams.set("x", x);
    url.searchParams.set("y", y);

    const res = await fetch(url);
    const tree = await res.json();

    ctx.clearRect(0, 0, canvas.width, canvas.height);
    renderNode(tree);
  }

  function renderNode(node) {
    if (!node) return;

    const { x, y, hw, hh } = node.boundary;

    ctx.strokeStyle = "white";
    ctx.strokeRect(x - hw, y - hh, hw * 2, hh * 2);

    if (node.points) {
      ctx.fillStyle = "lime";
      for (const p of node.points) {
        ctx.beginPath();
        ctx.arc(p.x, p.y, 2, 0, Math.PI * 2);
        ctx.fill();
      }
    }

    if (node.Divided) {
      renderNode(node.northEast);
      renderNode(node.northWest);
      renderNode(node.southEast);
      renderNode(node.southWest);
    }
  }

  let last = 0;

canvas.addEventListener("mousemove", (event) => {//suggested to use the click for more control over input 
  const now = performance.now();
  if (now - last < 100) return; // 1000/100 FPS
  last = now;

  const rect = canvas.getBoundingClientRect();
  const x = event.clientX - rect.left;
  const y = event.clientY - rect.top;

  loadAndRender(x, y);
});

