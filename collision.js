//it is to show that each particle checks each other to check wether they are colliding with them ,if it is then 
//high light all those colliding particle with self
 

let particles = [];
class Particle{
    constructor(x,y){
        this.x = x;
        this.y = y;
        this.r = 8;
    }

    move(){
        this.x += random(-1,1);
        this.y += random(-1,1);
    }

    render(){
        noStroke();
        fill(255);
        ellipse(this.x,this.y,this.r);
    }
}
function setup(){
    createCanvas(600,400);
    for (let i=0;i<100;i++){
        particles[i] = new Particle(random(width),random(height));

    }

}

function draw(){
    background(0);
    
    for ( let p of particles){
        p.move();
        p.render();
        
    }
}