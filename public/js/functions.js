async function submitData(form)
{
    let path = form.getAttribute("data-path")
    if(!path){
        console.log("Route cannot be found")
    }
    let result

    let data = new FormData(form)
    data = new URLSearchParams(data)
    path += "?"+data.toString()
    try{
        const response = await fetch(path, {
            method: "GET"
        })
        result = await response.json()
        if(!response.ok){
            if(result.error){
                throw new Error(result.error)
            } else{
                throw new Error("Une erreur a eut lieue")
            }
        }
        if(result.error){
            throw new Error(result.error)
        }
    } catch(error){
        return toast(error)
    }
    const divResult = document.querySelector("#content")
    divResult.innerHTML = result.content
    document.querySelector("#urlResult").innerHTML = '<a href="'+path+'">'+path+'</a>'
}

function toast(message, className="danger", duration=3000){
    Toastify({
        text: message,
        className: className,
        duration: duration,
        close: true,
        gravity: "top",
        position: "left",
        stopOnFocus: true,
    }).showToast();
}

function formHandler()
{
    const form = document.querySelector("#generateQR")
    if(!form){
        console.log("Form introuvable")
    }
    form.addEventListener("submit", function(e){
        e.preventDefault()
        submitData(form)
    })
}

function copyToClipboard(contentID){
    const scope = document.querySelector(contentID)
    if(!scope){
        return toast("Element not found")
    }
    scope.select()
    scope.setSelectionRange(0, 99999)
    navigator.clipboard.writeText(scope.value)
    return toast("Value copied to clipboard", "success")
}

formHandler()