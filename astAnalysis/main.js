import * as acorn from "acorn"
import * as recast from "recast";
import * as fs from 'node:fs';
const artfolder = "./artworks/"
const rawAstFolder = "astAnalysis/rawASTs/";

let acornconfig = {
    ecmaVersion: 9,
    sourceType: "script",
    allowReturnOutsideFunction: true
}

function main() {
      fs.readdir(artfolder, (err, files) => {
            files.forEach(file => {

                  const code = fs.readFileSync(artfolder + file).toString();
                  const ast = acorn.parse(code, acornconfig).body;
                  serializeAST(file, ast)
                  // AST transformation
                  // processAST(ast);
            });
      });
      
}

function serializeAST(file, ast) {
      const jsonString = JSON.stringify(ast, null, 2);
      fs.writeFileSync(rawAstFolder + file + ".json", jsonString, "utf8");
}

function processAST(ast) {
      // console.log(ast);
      let variableName;
      recast.visit(ast, {
            visitNode(path) {
                  const simplified = {
                        type: node.type,
                        children: []
                  };
                  this.traverse(path);
            }




            // visitVariableDeclaration(path) {
            //       console.log(path.value.type);
            //       this.traverse(path);
            // },
            // visitVariableDeclarator(path) {
            //       console.log("this is a declarator:" + path.value.id.name);                  
            //       this.traverse(path);
            // }
      });


      // recast.visit(
      //         ast,
      //         {
      //             visitVariableDeclaration: (path) => {
      //                   // variableName = path.node.callee.name;
      //                   console.log(path.type);
      //             }
      //       });
}

main();