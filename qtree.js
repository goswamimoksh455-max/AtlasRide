const canvas = document.getElementById("canvas");
const ctx = canvas.getContext("2d");

async function fetchAndDraw(){
    const response = await fetch('http://localhost:8080/data');
    const tree = await response.json();

    ctx.clearReact(0,0,600,600);
    drawNode(tree);
}

function drawNode(node){
    if (!node) return ;

    ctx.strokeStyle = '#444';
    ctx.strokeReact(
        node.boundary.x - node.boundary.hw,
        node.boundary.y - node.boundary.hh,
        node.boundary.hw*2,
        node.boundary.hh*2
    );

    ctx.fillStyle = '#00ffcc';
    node.points.forEach(p => {
        ctx.beginPath();
        ctx.arc(p.x,p.y,2,0,Math.PI*2);
        ctx.fill();
    });

    if(node.divided){
        drawNode(node.northEast);
        drawNode(node.northWest);
        drawNode(node.southEast);
        drawNode(node.southWest);
    }
}

fetchAndDraw();