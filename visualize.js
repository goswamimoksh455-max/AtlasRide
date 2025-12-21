
const canvas = document.getElementById("canvas");
const ctx = canvas.getContext("2d");

async function loadAndRender(){
    const url = new URL("http://localhost:8080/data");

    const res = await fetch(url);
    const qtree = await res.json();

    ctx.clearRect(0,0,canvas.width,canvas.height);
    renderNode(qtree)

}


function renderNode(node){
    if(!node){
        return ;

    }

    const {x, y, hw, hh }= node.boundary;

    ctx.strokeStyle = "white";
    ctx.strokeRect(x-hw, y-hh, hw*2, hh*2);

    if( node.points){
        ctx.fillStyle = "white";
        for ( const p of node.points){
            ctx.beginPath();
            ctx.arc(p.x,p.y,2,0,Math.PI*2);
            ctx.fill();
        }
    }

    if(node.Divided){
        renderNode(node.northEast);
        renderNode(node.northWest);
        renderNode(node.southEast);
        renderNode(node.southWest);
    }


}
var last = 0;
canvas.addEventListener("mousemove", async function (event) {
    const now = performance.now();
    if (now - last < 30) return;
    last = now;

    await loadAndRender();

    const rect = canvas.getBoundingClientRect();
    const x = event.clientX - rect.left;
    const y = event.clientY - rect.top;

    ctx.strokeStyle = "lime";
    ctx.strokeRect(x - 40, y - 40, 80, 80);

    const url = new URL("http://localhost:8080/query");
    url.searchParams.set("x", x);
    url.searchParams.set("y", y);
    url.searchParams.set("w", 40);
    url.searchParams.set("h", 40);

    const res = await fetch(url);
    const points = await res.json();

    ctx.fillStyle = "lime";
    points.pointsFound.forEach(p => {
        ctx.beginPath();
        ctx.arc(p.x, p.y, 2, 0, Math.PI * 2);
        ctx.fill();
    });
});

